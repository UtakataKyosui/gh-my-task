package ghclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
)

type Issue struct {
	Number    int       `json:"number"`
	Title     string    `json:"title"`
	URL       string    `json:"url"`
	Author    string    `json:"author"`
	State     string    `json:"state"`
	UpdatedAt time.Time `json:"updatedAt"`
	Labels    []string  `json:"labels"`
	Body      string    `json:"body,omitempty"`
}

type IssueOptions struct {
	State string // "open", "closed", "all"
}

func FetchIssues(owner, name string, opts IssueOptions) ([]Issue, error) {
	client, err := api.DefaultRESTClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}

	stateFilter := "is:open"
	if opts.State == "closed" {
		stateFilter = "is:closed"
	} else if opts.State == "all" {
		stateFilter = ""
	}

	repo := fmt.Sprintf("repo:%s/%s", owner, name)
	q := buildQuery("is:issue", stateFilter, repo, "assignee:@me")

	var resp searchResponse
	endpoint := "search/issues?per_page=100&q=" + url.QueryEscape(q)
	if err := client.Get(endpoint, &resp); err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	issues := make([]Issue, 0, len(resp.Items))
	for _, item := range resp.Items {
		t, _ := time.Parse(time.RFC3339, item.UpdatedAt)
		labels := make([]string, 0, len(item.Labels))
		for _, l := range item.Labels {
			labels = append(labels, l.Name)
		}
		issues = append(issues, Issue{
			Number:    item.Number,
			Title:     item.Title,
			URL:       item.HTMLURL,
			Author:    item.User.Login,
			State:     item.State,
			UpdatedAt: t,
			Labels:    labels,
			Body:      item.Body,
		})
	}

	sort.Slice(issues, func(i, j int) bool {
		return issues[i].UpdatedAt.After(issues[j].UpdatedAt)
	})

	return issues, nil
}

func CloseIssue(owner, name string, number int) error {
	client, err := api.DefaultRESTClient()
	if err != nil {
		return fmt.Errorf("failed to create API client: %w", err)
	}
	bodyBytes, err := json.Marshal(map[string]string{"state": "closed"})
	if err != nil {
		return err
	}
	var resp interface{}
	return client.Patch(
		fmt.Sprintf("repos/%s/%s/issues/%d", owner, name, number),
		bytes.NewReader(bodyBytes),
		&resp,
	)
}
