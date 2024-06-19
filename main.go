package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"

	"github.com/google/go-github/github"
	"github.com/joho/godotenv"
	"golang.org/x/oauth2"
)

type OrgRepos struct {
	OrgName string               `json:"org_name"`
	Repos   []*github.Repository `json:"repos"`
}

func helloWorldHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Hello, World!")
}

func fetchRepos(client *github.Client, orgName string, wg *sync.WaitGroup, reposChan chan<- OrgRepos) {
	defer wg.Done()
	opt := &github.RepositoryListByOrgOptions{Type: "public"}
	repos, _, err := client.Repositories.ListByOrg(context.Background(), orgName, opt)
	if err != nil {
		fmt.Printf("Error fetching repos for org %s: %v\n", orgName, err)
		return
	}
	reposChan <- OrgRepos{OrgName: orgName, Repos: repos}
}

func orgsHandler(w http.ResponseWriter, r *http.Request) {
	err := godotenv.Load()
	if err != nil {
		http.Error(w, "Error loading .env file", http.StatusInternalServerError)
		return
	}

	githubToken := os.Getenv("GITHUB_TOKEN")
	if githubToken == "" {
		http.Error(w, "GITHUB_TOKEN is not set in .env file", http.StatusInternalServerError)
		return
	}

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: githubToken},
	)
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)
	orgs, _, err := client.Organizations.List(ctx, "Shaurya0108", nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var wg sync.WaitGroup
	reposChan := make(chan OrgRepos, len(orgs))

	for _, org := range orgs {
		wg.Add(1)
		go fetchRepos(client, org.GetLogin(), &wg, reposChan)
	}

	wg.Wait()
	close(reposChan)

	var allOrgRepos []OrgRepos
	for orgRepos := range reposChan {
		allOrgRepos = append(allOrgRepos, orgRepos)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(allOrgRepos); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func main() {
	http.HandleFunc("/", helloWorldHandler)
	http.HandleFunc("/orgs", orgsHandler)
	fmt.Println("Starting server at port 8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Println(err)
	}
}
