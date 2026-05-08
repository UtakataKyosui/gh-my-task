package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"

	"github.com/UtakataKyosui/gh-my-task/internal/ghclient"
	"github.com/UtakataKyosui/gh-my-task/internal/install"
	"github.com/UtakataKyosui/gh-my-task/internal/jsonout"
	"github.com/UtakataKyosui/gh-my-task/internal/review/prompt"
	"github.com/UtakataKyosui/gh-my-task/internal/review/render"
	"github.com/UtakataKyosui/gh-my-task/internal/review/schema"
	"github.com/UtakataKyosui/gh-my-task/internal/review/validator"
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
	if len(os.Args) > 1 && os.Args[1] == "review" {
		runReview(os.Args[2:])
		return
	}

	var (
		jsonMode    bool
		state       string
		authorOnly  bool
		reviewOnly  bool
		includeDraft bool
		withIssues  bool
		issuesOnly  bool
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
	flag.BoolVar(&withIssues, "with-issues", false, "also show assigned issues alongside PRs")
	flag.BoolVar(&withIssues, "I", false, "also show assigned issues alongside PRs")
	flag.BoolVar(&issuesOnly, "issues-only", false, "show only assigned issues (no PRs)")

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

	if issuesOnly {
		withIssues = true
	}

	repo, err := repository.Current()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error: not in a GitHub repository (no remote found)")
		fmt.Fprintln(os.Stderr, "  hint: run this command from inside a git repository with a GitHub remote")
		os.Exit(1)
	}

	owner := repo.Owner
	name := repo.Name

	var prs []ghclient.PR
	if !issuesOnly {
		opts := ghclient.Options{
			State:        state,
			AuthorOnly:   authorOnly,
			ReviewOnly:   reviewOnly,
			IncludeDraft: includeDraft,
			WithReviews:  withReviews,
		}
		prs, err = ghclient.Fetch(owner, name, opts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: failed to fetch PRs: %v\n", err)
			os.Exit(1)
		}
	}

	var issues []ghclient.Issue
	if withIssues {
		issueOpts := ghclient.IssueOptions{State: state}
		issues, err = ghclient.FetchIssues(owner, name, issueOpts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: failed to fetch issues: %v\n", err)
			os.Exit(1)
		}
	}

	if jsonMode {
		user := ghclient.CurrentUser()
		if err := jsonout.Print(owner, name, user, prs, issues); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if len(prs) == 0 && len(issues) == 0 {
		fmt.Printf("No tasks found in %s/%s\n", owner, name)
		return
	}

	if err := tui.Run(owner, name, prs, issues); err != nil {
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

func runReview(args []string) {
	if len(args) == 0 {
		reviewUsage()
		os.Exit(1)
	}
	switch args[0] {
	case "schema":
		fmt.Println(string(schema.SchemaJSON))
	case "prompt":
		runReviewPrompt(args[1:])
	case "validate":
		runReviewValidate(args[1:])
	case "post":
		runReviewPost(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "unknown review subcommand: %s\n", args[0])
		reviewUsage()
		os.Exit(1)
	}
}

func reviewUsage() {
	fmt.Fprintln(os.Stderr, `gh my-task review — structured PR review with mandatory suggestions

USAGE:
  gh my-task review schema                      Print JSON Schema to stdout
  gh my-task review prompt <PR>                 Generate Claude prompt for a PR
  gh my-task review validate -f <file> [flags]  Validate review JSON without posting
  gh my-task review post <PR> -f <file> [flags] Validate and post review to GitHub

FLAGS (validate / post):
  -f string        Path to review JSON file (required)
  --pr int         PR number for diff-range validation (validate only)
  --min-comments   Override minimum comment count
  --strict         Treat warnings as errors
  --dry-run        (post only) Print API payload without posting`)
}

func runReviewPrompt(args []string) {
	fs := flag.NewFlagSet("review prompt", flag.ExitOnError)
	fs.Usage = func() { fmt.Fprintln(os.Stderr, "usage: gh my-task review prompt <PR>") }
	fs.Parse(args)

	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "error: PR番号を指定してください")
		os.Exit(1)
	}
	prNum, err := strconv.Atoi(fs.Arg(0))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: invalid PR number: %s\n", fs.Arg(0))
		os.Exit(1)
	}

	client, err := ghclient.NewReviewClient()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	pr, err := client.GetPR(prNum)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	files, err := client.GetFiles(prNum)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	p, err := prompt.Build(client.Owner(), client.Repo(), prompt.PRData{
		Number:       prNum,
		Title:        pr.Title,
		Author:       pr.User.Login,
		ChangedFiles: pr.ChangedFiles,
		Diff:         ghclient.BuildDiff(files),
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	fmt.Print(p)
}

func runReviewValidate(args []string) {
	fs := flag.NewFlagSet("review validate", flag.ExitOnError)
	file := fs.String("f", "", "review JSON file path")
	prNum := fs.Int("pr", 0, "PR number for diff-range validation")
	minComments := fs.Int("min-comments", 0, "override minimum comment count")
	strict := fs.Bool("strict", false, "treat warnings as errors")
	fs.Parse(args)

	if *file == "" {
		fmt.Fprintln(os.Stderr, "error: missing required flag: -f <review.json>")
		os.Exit(1)
	}

	rev, err := loadReview(*file)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	opts := validator.Options{MinComments: *minComments, Strict: *strict}
	if *prNum > 0 {
		client, err := ghclient.NewReviewClient()
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		pr, err := client.GetPR(*prNum)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		opts.ChangedFiles = pr.ChangedFiles
	}

	res := validator.Validate(rev, opts)
	printValidationResult(res)
	if !res.OK() {
		os.Exit(1)
	}
}

func runReviewPost(args []string) {
	fs := flag.NewFlagSet("review post", flag.ExitOnError)
	file := fs.String("f", "", "review JSON file path")
	minComments := fs.Int("min-comments", 0, "override minimum comment count")
	strict := fs.Bool("strict", false, "treat warnings as errors")
	dryRun := fs.Bool("dry-run", false, "print API payload without posting")
	fs.Parse(args)

	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "error: usage: gh my-task review post <PR> -f <review.json>")
		os.Exit(1)
	}
	prNum, err := strconv.Atoi(fs.Arg(0))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: invalid PR number: %s\n", fs.Arg(0))
		os.Exit(1)
	}
	if *file == "" {
		fmt.Fprintln(os.Stderr, "error: missing required flag: -f <review.json>")
		os.Exit(1)
	}

	rev, err := loadReview(*file)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	client, err := ghclient.NewReviewClient()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	pr, err := client.GetPR(prNum)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	opts := validator.Options{
		MinComments:  *minComments,
		Strict:       *strict,
		ChangedFiles: pr.ChangedFiles,
	}
	res := validator.Validate(rev, opts)
	printValidationResult(res)
	if !res.OK() {
		os.Exit(1)
	}

	payload := render.Build(rev)

	if *dryRun {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(payload); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		return
	}

	resp, err := client.PostReview(prNum, payload)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	fmt.Printf("Review posted: %s\n", resp.HTMLURL)
}

func loadReview(path string) (*schema.Review, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return schema.Decode(data)
}

func printValidationResult(res *validator.Result) {
	for _, w := range res.Warnings {
		fmt.Fprintln(os.Stderr, "warning:", w)
	}
	for _, e := range res.Errors {
		fmt.Fprintln(os.Stderr, "error:", e)
	}
	if res.OK() {
		fmt.Fprintln(os.Stderr, "validation passed")
	}
}
