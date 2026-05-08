# PR Review Task

You are reviewing Pull Request #{{.Number}} in {{.Owner}}/{{.Repo}}.

## PR Information

- **Title**: {{.Title}}
- **Changed files**: {{.ChangedFiles}}
- **Author**: {{.Author}}

## Diff

```diff
{{.Diff}}
```

## Output Requirements

**CRITICAL**: Output ONLY a single JSON object. No explanations, no Markdown fences around the JSON, no preamble.

The JSON must strictly conform to this schema:

```json
{{.Schema}}
```

## Review Rules (mandatory)

These rules come from the project's `pr-review.md` and are enforced by `gh my-task review validate`:

1. **summary must NOT contain suggestion blocks** — the `summary` field is for prose description only. Code proposals go exclusively in `comments[].suggestion`.
2. **Every inline comment MUST have a suggestion** — set `suggestion` to the replacement code. If a concrete suggestion is genuinely impossible (e.g. architectural discussion), set `skip_suggestion: true` AND provide a non-empty `reason`.
3. **Verify line numbers against the diff** — only reference lines that appear in the diff hunks above. Wrong line numbers cause the entire review to fail.
4. **Minimum comment count** — you must produce at least {{.MinComments}} inline comment(s) for a PR with {{.ChangedFiles}} changed file(s).
5. **List all issues first in summary** — number each problem in the summary, then address each as an inline comment.
6. **skip_suggestion abuse** — if more than 50% of your comments lack suggestions, the validator will warn or reject. Prefer concrete code fixes.

## Example (do not copy verbatim)

```json
{
  "$schema": "gh-code-review/v1",
  "event": "REQUEST_CHANGES",
  "summary": "Found 2 issues:\n1. Missing null check at src/foo.go:42\n2. Unnecessary allocation in src/bar.go:10",
  "comments": [
    {
      "path": "src/foo.go",
      "line": 42,
      "body": "This will panic if user is nil.",
      "suggestion": "if user == nil {\n\treturn nil, ErrNotFound\n}\nreturn user.Name, nil"
    },
    {
      "path": "src/bar.go",
      "line": 10,
      "body": "Allocating a new slice every call; reuse the buffer.",
      "suggestion": "buf = buf[:0]"
    }
  ]
}
```

Now produce the JSON review for PR #{{.Number}}.
