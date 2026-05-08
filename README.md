# gh my-task

A `gh` CLI extension that lists PRs in the current repository where you are the **author** or have a **review requested**.

## Installation

```bash
gh extension install UtakataKyosui/gh-my-task
```

## Usage

```
gh my-task [flags]
```

Run from inside any GitHub repository directory.

### Flags

| Flag | Default | Description |
|---|---|---|
| `--json` | false | Output JSON instead of launching TUI |
| `--state` | `open` | PR state filter: `open`, `closed`, `all` |
| `--author-only` | false | Show only PRs you authored |
| `--review-only` | false | Show only PRs where review is requested from you |
| `--include-drafts` | true | Include draft PRs |

### Examples

```bash
# Interactive TUI (default)
gh my-task

# JSON output for scripting / AI use
gh my-task --json

# All closed PRs you authored
gh my-task --state closed --author-only --json

# Pipe to jq
gh my-task --json | jq '.prs[].title'
```

## TUI Controls

| Key | Action |
|---|---|
| `↑` / `k` | Move up |
| `↓` / `j` | Move down |
| `Enter` | Open selected PR in browser |
| `/` | Filter list |
| `q` / `Ctrl+C` | Quit |

## Scheduled Snapshots

Periodically write the PR list for a repository to `.git/my-tasks/current.json` using a self-contained background daemon — no system cron or launchd required.

### Quick start

```bash
# Register current repo and start the daemon (default interval: 5m)
cd /path/to/your-repo
gh my-task schedule add

# With custom interval and filter
gh my-task schedule add --interval 10m --review-only
```

After registration the daemon starts automatically, fetches PRs immediately, then repeats at the configured interval. Output:

```
<repo>/.git/my-tasks/current.json
```

The file has the same JSON shape as `gh my-task --json`:

```json
{
  "repository": "owner/name",
  "user": "your-login",
  "fetchedAt": "2026-05-07T12:00:00Z",
  "prs": [...]
}
```

### Schedule commands

| Command | Description |
|---|---|
| `gh my-task schedule add [flags]` | Register current repo; start daemon if not running |
| `gh my-task schedule list` | List all registered schedules with last/next run times |
| `gh my-task schedule remove` | Unregister current repo |
| `gh my-task schedule start` | Start daemon manually |
| `gh my-task schedule stop` | Stop daemon (SIGTERM) |
| `gh my-task schedule status` | Show daemon PID and schedule count |

### `schedule add` flags

| Flag | Default | Description |
|---|---|---|
| `--interval` | `5m` | Fetch interval (e.g. `1m`, `10m`, `1h`) |
| `--state` / `-s` | `open` | PR state filter |
| `--author-only` / `-a` | false | Only PRs you authored |
| `--review-only` / `-r` | false | Only PRs where review is requested |
| `--include-drafts` / `-d` | true | Include draft PRs |
| `--with-reviews` / `-R` | false | Fetch review status (slower) |

### File locations

| Path | Purpose |
|---|---|
| `<repo>/.git/my-tasks/current.json` | PR snapshot (inside `.git/`, not tracked by Git) |
| `$XDG_CONFIG_HOME/gh-my-task/schedules.json` | Registry of all schedules |
| `$XDG_CONFIG_HOME/gh-my-task/daemon.pid` | Running daemon PID |
| `$XDG_CONFIG_HOME/gh-my-task/daemon.log` | Daemon log |

(`$XDG_CONFIG_HOME` = `~/Library/Application Support` on macOS, `~/.config` on Linux)

### Auth

The daemon inherits `gh auth` from the parent shell via `go-gh`. If running in a stripped environment, set `GH_TOKEN`.

---

## PR Review (`review` subcommand)

Validate and post Claude-generated PR reviews with mandatory suggestion blocks.

### Workflow

```bash
# 1. Generate a Claude-ready prompt (includes PR diff + schema + review rules)
gh my-task review prompt 123 | claude --print > review.json

# 2. Validate without posting
gh my-task review validate -f review.json --pr 123

# 3. Post to GitHub
gh my-task review post 123 -f review.json
```

### Review subcommands

| Command | Description |
|---|---|
| `gh my-task review schema` | Print JSON Schema to stdout |
| `gh my-task review prompt <PR>` | Generate Claude-ready prompt for a PR |
| `gh my-task review validate -f <file>` | Validate review JSON without posting |
| `gh my-task review post <PR> -f <file>` | Validate and post review to GitHub |

### Review flags

| Flag | Commands | Description |
|---|---|---|
| `-f <file>` | validate, post | Review JSON file path (required) |
| `--pr <N>` | validate | PR number for diff-range validation |
| `--min-comments <N>` | validate, post | Override minimum comment count |
| `--strict` | validate, post | Treat warnings as errors |
| `--dry-run` | post | Print API payload without posting |

### Validation rules

| # | Rule |
|---|---|
| 1 | `event`, `summary`, `comments` are required |
| 2 | `summary` must not contain ` ```suggestion ` blocks |
| 3 | Minimum comment count scales with changed-file count (≤4→1, 5–20→3, 21+→5) |
| 4a | Each comment needs `suggestion` OR `skip_suggestion: true` + `reason` |
| 4b | `skip_suggestion: true` without `reason` is rejected |
| 4c | More than 50% skip_suggestion triggers a warning (error with `--strict`) |

---

## PR Badges

- `[A]` — You are the author
- `[R]` — Review is requested from you
- `[AR]` — Both
