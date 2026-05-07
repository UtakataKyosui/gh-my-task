package schedule

import (
	"encoding/json"
	"time"

	"github.com/UtakataKyosui/gh-my-task/internal/ghclient"
)

type Filter struct {
	State        string `json:"state"`
	AuthorOnly   bool   `json:"authorOnly"`
	ReviewOnly   bool   `json:"reviewOnly"`
	IncludeDraft bool   `json:"includeDraft"`
	WithReviews  bool   `json:"withReviews"`
	WithIssues   bool   `json:"withIssues"`
}

func (f Filter) ToOptions() ghclient.Options {
	return ghclient.Options{
		State:        f.State,
		AuthorOnly:   f.AuthorOnly,
		ReviewOnly:   f.ReviewOnly,
		IncludeDraft: f.IncludeDraft,
		WithReviews:  f.WithReviews,
	}
}

func (f Filter) ToIssueOptions() ghclient.IssueOptions {
	return ghclient.IssueOptions{State: f.State}
}

// duration wraps time.Duration to marshal/unmarshal as human-readable string (e.g. "5m").
type duration struct{ time.Duration }

func (d duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

func (d *duration) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	v, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	d.Duration = v
	return nil
}

type Schedule struct {
	ID        string    `json:"id"`
	RepoPath  string    `json:"repoPath"`
	Owner     string    `json:"owner"`
	Name      string    `json:"name"`
	Interval  duration  `json:"interval"`
	Filter    Filter    `json:"filter"`
	LastRun   time.Time `json:"lastRun,omitempty"`
	NextRun   time.Time `json:"nextRun,omitempty"`
	LastError string    `json:"lastError,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}
