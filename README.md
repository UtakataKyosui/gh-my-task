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

## PR Badges

- `[A]` — You are the author
- `[R]` — Review is requested from you
- `[AR]` — Both
