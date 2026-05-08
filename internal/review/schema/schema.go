package schema

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
)

//go:embed schema.json
var SchemaJSON []byte

type Event string

const (
	EventComment        Event = "COMMENT"
	EventRequestChanges Event = "REQUEST_CHANGES"
	EventApprove        Event = "APPROVE"
)

type Comment struct {
	Path           string `json:"path"`
	Line           int    `json:"line"`
	Side           string `json:"side,omitempty"`
	StartLine      int    `json:"start_line,omitempty"`
	Body           string `json:"body"`
	Suggestion     string `json:"suggestion,omitempty"`
	SkipSuggestion bool   `json:"skip_suggestion,omitempty"`
	Reason         string `json:"reason,omitempty"`
}

type Review struct {
	Schema   string    `json:"$schema,omitempty"`
	Event    Event     `json:"event"`
	Summary  string    `json:"summary"`
	Comments []Comment `json:"comments"`
}

func Decode(data []byte) (*Review, error) {
	var r Review
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("JSON parse error: %w", err)
	}
	return &r, nil
}

func (c *Comment) EffectiveSide() string {
	if c.Side == "" {
		return "RIGHT"
	}
	return strings.ToUpper(c.Side)
}
