package validator

import (
	"fmt"
	"strings"

	"github.com/UtakataKyosui/gh-my-task/internal/review/schema"
	"github.com/UtakataKyosui/gh-my-task/internal/review/threshold"
)

type Options struct {
	MinComments  int
	Strict       bool
	ChangedFiles int
}

type Result struct {
	Errors   []string
	Warnings []string
}

func (r *Result) OK() bool {
	return len(r.Errors) == 0
}

func Validate(rev *schema.Review, opts Options) *Result {
	res := &Result{}

	if rev.Event == "" {
		res.Errors = append(res.Errors, "[1] missing required field: event")
	}
	if rev.Summary == "" {
		res.Errors = append(res.Errors, "[1] missing required field: summary")
	}
	if rev.Comments == nil {
		res.Errors = append(res.Errors, "[1] missing required field: comments")
	}

	if strings.Contains(rev.Summary, "```suggestion") {
		res.Errors = append(res.Errors, "[2] summary must not contain ```suggestion blocks — code suggestions belong in inline comments only")
	}

	if len(rev.Comments) > 0 || opts.ChangedFiles > 0 {
		minRequired := opts.MinComments
		if minRequired <= 0 && opts.ChangedFiles > 0 {
			minRequired = threshold.MinComments(opts.ChangedFiles)
		}
		if minRequired > 0 && len(rev.Comments) < minRequired {
			res.Errors = append(res.Errors, fmt.Sprintf(
				"[3] too few inline comments: got %d, need at least %d (PR has %d changed files)",
				len(rev.Comments), minRequired, opts.ChangedFiles,
			))
		}
	}

	skipCount := 0
	for i, c := range rev.Comments {
		idx := i + 1
		hasSuggestion := strings.TrimSpace(c.Suggestion) != ""
		if c.SkipSuggestion {
			skipCount++
			if strings.TrimSpace(c.Reason) == "" {
				res.Errors = append(res.Errors, fmt.Sprintf(
					"[4] comment #%d: skip_suggestion is true but reason is empty", idx,
				))
			}
		} else if !hasSuggestion {
			res.Errors = append(res.Errors, fmt.Sprintf(
				"[4] comment #%d (%s:%d): missing suggestion — either provide a suggestion or set skip_suggestion:true with reason",
				idx, c.Path, c.Line,
			))
		}
	}

	if len(rev.Comments) > 0 {
		ratio := float64(skipCount) / float64(len(rev.Comments))
		if ratio > 0.5 {
			msg := fmt.Sprintf(
				"[4] %d of %d comments use skip_suggestion (%.0f%%) — more than half lack concrete suggestions",
				skipCount, len(rev.Comments), ratio*100,
			)
			if opts.Strict {
				res.Errors = append(res.Errors, msg)
			} else {
				res.Warnings = append(res.Warnings, msg)
			}
		}
	}

	if opts.Strict && len(res.Warnings) > 0 {
		res.Errors = append(res.Errors, res.Warnings...)
		res.Warnings = nil
	}

	return res
}
