package schedule

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/cli/go-gh/v2/pkg/repository"
)

// Dispatch routes `gh my-task schedule <subcommand> [args]`.
func Dispatch(args []string) {
	if len(args) == 0 {
		printScheduleUsage()
		os.Exit(1)
	}
	switch args[0] {
	case "add":
		RunAdd(args[1:])
	case "list":
		RunList(args[1:])
	case "remove", "rm":
		RunRemove(args[1:])
	case "start":
		RunStart(args[1:])
	case "stop":
		RunStop(args[1:])
	case "status":
		RunStatus(args[1:])
	case "_run-daemon":
		RunDaemon()
	default:
		fmt.Fprintf(os.Stderr, "unknown schedule subcommand: %q\n", args[0])
		printScheduleUsage()
		os.Exit(1)
	}
}

func printScheduleUsage() {
	fmt.Fprintln(os.Stderr, `usage: gh my-task schedule <subcommand> [flags]

subcommands:
  add      Register current repository and start the daemon
  list     Show all registered schedules
  remove   Unregister current repository
  start    Start the daemon manually
  stop     Stop the daemon
  status   Show daemon status`)
}

// RunAdd registers the current repo and ensures the daemon is running.
func RunAdd(args []string) {
	fs := flag.NewFlagSet("schedule add", flag.ExitOnError)
	var (
		intervalStr  string
		state        string
		authorOnly   bool
		reviewOnly   bool
		includeDraft bool
		withReviews  bool
	)
	fs.StringVar(&intervalStr, "interval", "5m", "fetch interval (e.g. 1m, 10m, 1h)")
	fs.StringVar(&state, "state", "open", "PR state: open, closed, all")
	fs.StringVar(&state, "s", "open", "PR state: open, closed, all")
	fs.BoolVar(&authorOnly, "author-only", false, "only PRs you authored")
	fs.BoolVar(&authorOnly, "a", false, "only PRs you authored")
	fs.BoolVar(&reviewOnly, "review-only", false, "only PRs where review is requested")
	fs.BoolVar(&reviewOnly, "r", false, "only PRs where review is requested")
	fs.BoolVar(&includeDraft, "include-drafts", true, "include draft PRs")
	fs.BoolVar(&includeDraft, "d", true, "include draft PRs")
	fs.BoolVar(&withReviews, "with-reviews", false, "fetch review status (slower)")
	fs.BoolVar(&withReviews, "R", false, "fetch review status (slower)")
	_ = fs.Parse(args)

	interval, err := time.ParseDuration(intervalStr)
	if err != nil || interval <= 0 {
		fmt.Fprintf(os.Stderr, "error: invalid interval %q\n", intervalStr)
		os.Exit(1)
	}
	if authorOnly && reviewOnly {
		fmt.Fprintln(os.Stderr, "error: --author-only and --review-only are mutually exclusive")
		os.Exit(1)
	}

	repo, err := repository.Current()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error: not in a GitHub repository")
		os.Exit(1)
	}

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error: cannot determine current directory")
		os.Exit(1)
	}
	absPath, err := filepath.Abs(cwd)
	if err != nil {
		absPath = cwd
	}

	s := Schedule{
		RepoPath:  absPath,
		Owner:     repo.Owner,
		Name:      repo.Name,
		Interval:  duration{interval},
		Filter: Filter{
			State:        state,
			AuthorOnly:   authorOnly,
			ReviewOnly:   reviewOnly,
			IncludeDraft: includeDraft,
			WithReviews:  withReviews,
		},
		CreatedAt: time.Now().UTC(),
	}

	reg, err := LoadRegistry()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: cannot load registry: %v\n", err)
		os.Exit(1)
	}
	reg.Add(s)
	if err := reg.Save(); err != nil {
		fmt.Fprintf(os.Stderr, "error: cannot save registry: %v\n", err)
		os.Exit(1)
	}

	if err := EnsureDaemon(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to start/reload daemon: %v\n", err)
	}

	outPath := RepoOutputPath(absPath)
	fmt.Printf("registered %s/%s (interval: %s)\n", repo.Owner, repo.Name, interval)
	fmt.Printf("output: %s\n", outPath)
}

// RunList prints all registered schedules.
func RunList(_ []string) {
	reg, err := LoadRegistry()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	schedules := reg.Snapshot()
	if len(schedules) == 0 {
		fmt.Println("no schedules registered")
		return
	}
	pid, running := DaemonPID()
	if running {
		fmt.Printf("daemon running (pid %d)\n\n", pid)
	} else {
		fmt.Println("daemon not running")
	fmt.Println()
	}
	for _, s := range schedules {
		lastRun := "never"
		if !s.LastRun.IsZero() {
			lastRun = s.LastRun.Local().Format("2006-01-02 15:04:05")
		}
		nextRun := "-"
		if !s.NextRun.IsZero() {
			nextRun = s.NextRun.Local().Format("2006-01-02 15:04:05")
		}
		errSuffix := ""
		if s.LastError != "" {
			errSuffix = fmt.Sprintf("  [error: %s]", s.LastError)
		}
		fmt.Printf("  %s/%s  every %s  last: %s  next: %s%s\n",
			s.Owner, s.Name, s.Interval, lastRun, nextRun, errSuffix)
	}
}

// RunRemove unregisters the current repo's schedule.
func RunRemove(_ []string) {
	repo, err := repository.Current()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error: not in a GitHub repository")
		os.Exit(1)
	}

	cwd, _ := os.Getwd()
	absPath, _ := filepath.Abs(cwd)

	reg, err := LoadRegistry()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	if !reg.Remove(absPath) {
		fmt.Fprintf(os.Stderr, "no schedule found for %s/%s\n", repo.Owner, repo.Name)
		os.Exit(1)
	}
	if err := reg.Save(); err != nil {
		fmt.Fprintf(os.Stderr, "error: cannot save registry: %v\n", err)
		os.Exit(1)
	}

	// Signal daemon to reload (stops the runner for this repo).
	if err := EnsureDaemon(); err != nil {
		// Daemon may not be running — that's fine.
	}
	fmt.Printf("removed schedule for %s/%s\n", repo.Owner, repo.Name)
}

// RunStart starts the daemon if not already running.
func RunStart(_ []string) {
	if pid, running := DaemonPID(); running {
		fmt.Printf("daemon already running (pid %d)\n", pid)
		return
	}
	if err := spawnDaemon(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("daemon started")
}

// RunStop stops the daemon.
func RunStop(_ []string) {
	if err := StopDaemon(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("daemon stopped")
}

// RunStatus prints daemon status.
func RunStatus(_ []string) {
	pid, running := DaemonPID()
	reg, _ := LoadRegistry()

	count := 0
	if reg != nil {
		count = len(reg.Snapshot())
	}
	logPath, _ := LogPath()

	if running {
		fmt.Printf("daemon running  pid=%d  schedules=%d  log=%s\n", pid, count, logPath)
	} else {
		fmt.Printf("daemon not running  schedules=%d  log=%s\n", count, logPath)
	}
}
