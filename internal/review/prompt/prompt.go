package prompt

import (
	"bytes"
	_ "embed"
	"fmt"
	"text/template"

	"github.com/UtakataKyosui/gh-my-task/internal/review/schema"
	"github.com/UtakataKyosui/gh-my-task/internal/review/threshold"
)

//go:embed template.md
var tmplSrc string

var tmpl = template.Must(template.New("prompt").Parse(tmplSrc))

type PRData struct {
	Number       int
	Title        string
	Author       string
	ChangedFiles int
	Diff         string
}

type prContext struct {
	Number       int
	Owner        string
	Repo         string
	Title        string
	Author       string
	ChangedFiles int
	Diff         string
	MinComments  int
	Schema       string
}

const defaultMaxDiffChars = 80_000

func truncateDiff(diff string, maxChars int) (string, bool) {
	if len(diff) <= maxChars {
		return diff, false
	}
	cut := diff[:maxChars]
	if idx := len(cut) - 1; idx >= 0 {
		for idx > 0 && cut[idx] != '\n' {
			idx--
		}
		cut = diff[:idx+1]
	}
	omitted := len(diff) - len(cut)
	return cut + fmt.Sprintf("\n[diff truncated — %d chars omitted]\n", omitted), true
}

func Build(owner, repo string, pr PRData) (string, error) {
	minComments := threshold.MinComments(pr.ChangedFiles)
	diff, _ := truncateDiff(pr.Diff, defaultMaxDiffChars)
	ctx := prContext{
		Number:       pr.Number,
		Owner:        owner,
		Repo:         repo,
		Title:        pr.Title,
		Author:       pr.Author,
		ChangedFiles: pr.ChangedFiles,
		MinComments:  minComments,
		Diff:         diff,
		Schema:       string(schema.SchemaJSON),
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, ctx); err != nil {
		return "", fmt.Errorf("prompt template execute: %w", err)
	}
	return buf.String(), nil
}
