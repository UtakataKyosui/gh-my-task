package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"

	"github.com/UtakataKyosui/gh-my-task/internal/ghclient"
	"github.com/UtakataKyosui/gh-my-task/internal/install"
	"github.com/UtakataKyosui/gh-my-task/internal/jsonout"
	"github.com/UtakataKyosui/gh-my-task/internal/schedule"
	"github.com/UtakataKyosui/gh-my-task/internal/tui"
	"github.com/cli/go-gh/v2/pkg/repository"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "close" {
		runClose(os.Args[2:])
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "prompt" {
		runPrompt(os.Args[2:])
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "install" {
		install.Run()
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "schedule" {
		schedule.Dispatch(os.Args[2:])
		return
	}

	var (
		jsonMode     bool
		state        string
		authorOnly   bool
		reviewOnly   bool
		includeDraft bool
	)

	flag.BoolVar(&jsonMode, "json", false, "output JSON instead of TUI")
	flag.BoolVar(&jsonMode, "j", false, "output JSON instead of TUI")
	flag.StringVar(&state, "state", "open", "PR state: open, closed, all")
	flag.StringVar(&state, "s", "open", "PR state: open, closed, all")
	flag.BoolVar(&authorOnly, "author-only", false, "show only PRs you authored")
	flag.BoolVar(&authorOnly, "a", false, "show only PRs you authored")
	flag.BoolVar(&reviewOnly, "review-only", false, "show only PRs where review is requested")
	flag.BoolVar(&reviewOnly, "r", false, "show only PRs where review is requested")
	flag.BoolVar(&includeDraft, "include-drafts", true, "include draft PRs")
	flag.BoolVar(&includeDraft, "d", true, "include draft PRs")

	var withReviews bool
	flag.BoolVar(&withReviews, "with-reviews", false, "fetch review status for each PR (slower)")
	flag.BoolVar(&withReviews, "R", false, "fetch review status for each PR (slower)")

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
		WithReviews:  withReviews,
	}

	prs, err := ghclient.Fetch(owner, name, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to fetch PRs: %v\n", err)
		os.Exit(1)
	}

	if jsonMode {
		user := ghclient.CurrentUser()
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

func runPrompt(args []string) {
	fs := flag.NewFlagSet("prompt", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: gh my-task prompt <PR-number>")
	}
	_ = fs.Parse(args)

	if fs.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "error: PR番号を指定してください")
		fmt.Fprintln(os.Stderr, "  usage: gh my-task prompt <PR-number>")
		os.Exit(1)
	}

	number, err := strconv.Atoi(fs.Arg(0))
	if err != nil || number <= 0 {
		fmt.Fprintf(os.Stderr, "error: 無効なPR番号 %q\n", fs.Arg(0))
		os.Exit(1)
	}

	repo, err := repository.Current()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error: not in a GitHub repository (no remote found)")
		os.Exit(1)
	}

	pr, err := ghclient.FetchOne(repo.Owner, repo.Name, number)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: PR #%d の取得に失敗しました: %v\n", number, err)
		os.Exit(1)
	}

	fmt.Print(ghclient.BuildPrompt(repo.Owner, repo.Name, pr))
}

func runClose(args []string) {
	fs := flag.NewFlagSet("close", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: gh my-task close <PR-number>")
	}
	_ = fs.Parse(args)

	if fs.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "error: PR番号を指定してください")
		fmt.Fprintln(os.Stderr, "  usage: gh my-task close <PR-number>")
		os.Exit(1)
	}

	number, err := strconv.Atoi(fs.Arg(0))
	if err != nil || number <= 0 {
		fmt.Fprintf(os.Stderr, "error: 無効なPR番号 %q\n", fs.Arg(0))
		os.Exit(1)
	}

	repo, err := repository.Current()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error: not in a GitHub repository (no remote found)")
		os.Exit(1)
	}

	owner := repo.Owner
	name := repo.Name

	pr, err := ghclient.FetchOne(owner, name, number)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: PR #%d の取得に失敗しました: %v\n", number, err)
		os.Exit(1)
	}

	fmt.Printf("PR #%d: %s\n", pr.Number, pr.Title)
	fmt.Printf("Author: %s  State: %s\n", pr.Author, pr.State)
	fmt.Printf("URL: %s\n\n", pr.URL)
	fmt.Printf("この PR を close しますか？ 確認のため PR 番号を入力してください: ")

	var input string
	_, _ = fmt.Scan(&input)

	if input != strconv.Itoa(number) {
		fmt.Println("abort: 番号が一致しないため、close をキャンセルしました")
		os.Exit(0)
	}

	if err := ghclient.ClosePR(owner, name, number); err != nil {
		fmt.Fprintf(os.Stderr, "error: PR #%d の close に失敗しました: %v\n", number, err)
		os.Exit(1)
	}

	fmt.Printf("✓ PR #%d を close しました\n", number)
}

