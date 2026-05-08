package render

import (
	"fmt"
	"strings"

	"github.com/UtakataKyosui/gh-my-task/internal/review/schema"
)

type CommentPayload struct {
	Path      string `json:"path"`
	Line      int    `json:"line"`
	Side      string `json:"side"`
	StartLine int    `json:"start_line,omitempty"`
	Body      string `json:"body"`
}

type ReviewPayload struct {
	Body     string           `json:"body"`
	Event    string           `json:"event"`
	Comments []CommentPayload `json:"comments"`
}

func Build(rev *schema.Review) *ReviewPayload {
	comments := make([]CommentPayload, 0, len(rev.Comments))
	for _, c := range rev.Comments {
		comments = append(comments, renderComment(c))
	}
	return &ReviewPayload{
		Body:     rev.Summary,
		Event:    string(rev.Event),
		Comments: comments,
	}
}

func renderComment(c schema.Comment) CommentPayload {
	body := buildBody(c)
	p := CommentPayload{
		Path: c.Path,
		Line: c.Line,
		Side: c.EffectiveSide(),
		Body: body,
	}
	if c.StartLine > 0 && c.StartLine < c.Line {
		p.StartLine = c.StartLine
	}
	return p
}

func buildBody(c schema.Comment) string {
	var sb strings.Builder
	sb.WriteString(c.Body)

	if strings.TrimSpace(c.Suggestion) != "" {
		sb.WriteString("\n\n```suggestion\n")
		sb.WriteString(c.Suggestion)
		if !strings.HasSuffix(c.Suggestion, "\n") {
			sb.WriteByte('\n')
		}
		sb.WriteString("```")
	} else if c.SkipSuggestion && c.Reason != "" {
		sb.WriteString(fmt.Sprintf("\n\n<!-- skip-reason: %s -->", c.Reason))
	}

	return sb.String()
}
