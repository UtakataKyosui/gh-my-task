package ghclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/UtakataKyosui/gh-my-task/internal/review/render"
	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
)

// ReviewClient wraps REST calls for PR review operations.
type ReviewClient struct {
	rest  *api.RESTClient
	owner string
	repo  string
}

func NewReviewClient() (*ReviewClient, error) {
	rest, err := api.DefaultRESTClient()
	if err != nil {
		return nil, fmt.Errorf("gh auth error: %w\nRun: gh auth login", err)
	}
	repo, err := repository.Current()
	if err != nil {
		return nil, fmt.Errorf("could not determine repository; run from a git repo: %w", err)
	}
	return &ReviewClient{rest: rest, owner: repo.Owner, repo: repo.Name}, nil
}

func (c *ReviewClient) repoPath(path string) string {
	return fmt.Sprintf("repos/%s/%s/%s", c.owner, c.repo, path)
}

func (c *ReviewClient) Owner() string { return c.owner }
func (c *ReviewClient) Repo() string  { return c.repo }

type PRMeta struct {
	ChangedFiles int    `json:"changed_files"`
	Number       int    `json:"number"`
	Title        string `json:"title"`
	User         struct {
		Login string `json:"login"`
	} `json:"user"`
}

type FilePatch struct {
	Filename string `json:"filename"`
	Patch    string `json:"patch"`
}

func (c *ReviewClient) GetPR(number int) (*PRMeta, error) {
	var pr PRMeta
	if err := c.rest.Get(c.repoPath(fmt.Sprintf("pulls/%d", number)), &pr); err != nil {
		return nil, fmt.Errorf("get PR #%d: %w", number, err)
	}
	return &pr, nil
}

func (c *ReviewClient) GetFiles(prNumber int) ([]FilePatch, error) {
	var files []FilePatch
	if err := c.rest.Get(c.repoPath(fmt.Sprintf("pulls/%d/files", prNumber)), &files); err != nil {
		return nil, fmt.Errorf("get PR #%d files: %w", prNumber, err)
	}
	return files, nil
}

func BuildDiff(files []FilePatch) string {
	var sb strings.Builder
	for _, f := range files {
		if f.Patch == "" {
			continue
		}
		sb.WriteString("diff --git a/")
		sb.WriteString(f.Filename)
		sb.WriteString(" b/")
		sb.WriteString(f.Filename)
		sb.WriteByte('\n')
		sb.WriteString(f.Patch)
		sb.WriteByte('\n')
	}
	return sb.String()
}

type ReviewResponse struct {
	ID      int    `json:"id"`
	State   string `json:"state"`
	HTMLURL string `json:"html_url"`
}

func (c *ReviewClient) PostReview(prNumber int, payload *render.ReviewPayload) (*ReviewResponse, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal review payload: %w", err)
	}
	var resp ReviewResponse
	if err := c.rest.Post(
		c.repoPath(fmt.Sprintf("pulls/%d/reviews", prNumber)),
		bytes.NewReader(body),
		&resp,
	); err != nil {
		return nil, fmt.Errorf("post review to PR #%d: %w", prNumber, err)
	}
	return &resp, nil
}
