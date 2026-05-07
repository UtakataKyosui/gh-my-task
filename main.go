package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/UtakataKyosui/gh-my-task/internal/ghclient"
	"github.com/UtakataKyosui/gh-my-task/internal/jsonout"
	"github.com/UtakataKyosui/gh-my-task/internal/tui"
	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
)

func main() {
	var (
		jsonMode     bool
		state        string
		authorOnly   bool
		reviewOnly   bool
		includeDraft bool
	)

	flag.BoolVar(&jsonMode, "json", false, "output JSON instead of TUI")
	flag.StringVar(&state, "state", "open", "PR state: open, closed, all")
	flag.BoolVar(&authorOnly, "author-only", false, "show only PRs you authored")
	flag.BoolVar(&reviewOnly, "review-only", false, "show only PRs where review is requested")
	flag.BoolVar(&includeDraft, "include-drafts", true, "include draft PRs")
	flag.Parse()

	if authorOnly && reviewOnly {
		fmt.Fprintln(os.Stderr, "error: --author-only and --review-only are mutually exclusive")
		os.Exit(1)
	}

	if state != "open" && state != "closed" && state != "all" {
		fmt.Fprintf(os.Stderr, "error: --state must be open, closed, or all (got %q)\n", state)
		os.Exit(1)
	}

	repo, err := repository.Current()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error: not in a GitHub repository (no remote found)")
		fmt.Fprintln(os.Stderr, "  hint: run this command from inside a git repository with a GitHub remote")
		os.Exit(1)
	}

	owner := repo.Owner
	name := repo.Name

	opts := ghclient.Options{
		State:        state,
		AuthorOnly:   authorOnly,
		ReviewOnly:   reviewOnly,
		IncludeDraft: includeDraft,
	}

	prs, err := ghclient.Fetch(owner, name, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to fetch PRs: %v\n", err)
		os.Exit(1)
	}

	if jsonMode {
		user := currentUser()
		if err := jsonout.Print(owner, name, user, prs); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if len(prs) == 0 {
		fmt.Printf("No PRs found in %s/%s\n", owner, name)
		return
	}

	if err := tui.Run(owner, name, prs); err != nil {
		fmt.Fprintf(os.Stderr, "error: TUI failed: %v\n", err)
		os.Exit(1)
	}
}

func currentUser() string {
	client, err := api.DefaultRESTClient()
	if err != nil {
		return ""
	}
	var resp struct {
		Login string `json:"login"`
	}
	if err := client.Get("user", &resp); err != nil {
		return ""
	}
	return resp.Login
}
