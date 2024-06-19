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

// OrgRepos is a struct to hold organization name and its repositories.
type OrgRepos struct {
	OrgName string               `json:"org_name"`
	Repos   []*github.Repository `json:"repos"`
}

// helloWorldHandler responds with "Hello, World!" when accessed at the root endpoint.
func helloWorldHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Hello, World!")
}

// fetchRepos fetches public repositories for a given organization and sends the result to reposChan.
func fetchRepos(client *github.Client, orgName string, wg *sync.WaitGroup, reposChan chan<- OrgRepos) {
	defer wg.Done() // Decrement the wait group counter when the function returns.

	// Set the options to list only public repositories.
	opt := &github.RepositoryListByOrgOptions{Type: "public"}

	// Fetch the repositories for the organization.
	repos, _, err := client.Repositories.ListByOrg(context.Background(), orgName, opt)
	if err != nil {
		fmt.Printf("Error fetching repos for org %s: %v\n", orgName, err)
		return
	}

	// Send the organization name and its repositories to the channel.
	reposChan <- OrgRepos{OrgName: orgName, Repos: repos}
}

// orgsHandler handles the /orgs endpoint, fetching organizations and their repositories.
func orgsHandler(w http.ResponseWriter, r *http.Request) {
	// Load the .env file to get the environment variables.
	err := godotenv.Load()
	if err != nil {
		http.Error(w, "Error loading .env file", http.StatusInternalServerError)
		return
	}

	// Get the GitHub token from the environment variables.
	githubToken := os.Getenv("GITHUB_TOKEN")
	if githubToken == "" {
		http.Error(w, "GITHUB_TOKEN is not set in .env file", http.StatusInternalServerError)
		return
	}

	// Create a context and an OAuth2 client using the GitHub token.
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: githubToken},
	)
	tc := oauth2.NewClient(ctx, ts)

	// Create a GitHub client.
	client := github.NewClient(tc)

	// Fetch the list of organizations for the user "Shaurya0108".
	orgs, _, err := client.Organizations.List(ctx, "Shaurya0108", nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Use a wait group to wait for all goroutines to finish.
	var wg sync.WaitGroup

	// Create a channel to receive the results.
	reposChan := make(chan OrgRepos, len(orgs))

	// Start a goroutine for each organization to fetch its repositories.
	for _, org := range orgs {
		wg.Add(1)
		go fetchRepos(client, org.GetLogin(), &wg, reposChan)
	}

	// Wait for all goroutines to finish.
	wg.Wait()
	close(reposChan)

	// Collect the results from the channel.
	var allOrgRepos []OrgRepos
	for orgRepos := range reposChan {
		allOrgRepos = append(allOrgRepos, orgRepos)
	}

	// Set the response header to indicate JSON content.
	w.Header().Set("Content-Type", "application/json")

	// Encode the results as JSON and write them to the response.
	if err := json.NewEncoder(w).Encode(allOrgRepos); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func main() {
	// Register the handler functions for the root and /orgs endpoints.
	http.HandleFunc("/", helloWorldHandler)
	http.HandleFunc("/orgs", orgsHandler)

	// Start the HTTP server on port 8080.
	fmt.Println("Starting server at port 8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Println(err)
	}
}
