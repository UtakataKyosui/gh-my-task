package ghclient

import (
	"fmt"
	"net/url"
	"sort"
	"sync"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
)

type PR struct {
	Number     int       `json:"number"`
	Title      string    `json:"title"`
	URL        string    `json:"url"`
	Author     string    `json:"author"`
	State      string    `json:"state"`
	IsDraft    bool      `json:"isDraft"`
	UpdatedAt  time.Time `json:"updatedAt"`
	Labels     []string  `json:"labels"`
	Categories []string  `json:"categories"`
	Body       string    `json:"body,omitempty"`
}

type searchResponse struct {
	Items []searchItem `json:"items"`
}

type searchItem struct {
	Number    int    `json:"number"`
	Title     string `json:"title"`
	HTMLURL   string `json:"html_url"`
	State     string `json:"state"`
	Draft     bool   `json:"draft"`
	UpdatedAt string `json:"updated_at"`
	Body      string `json:"body"`
	User      struct {
		Login string `json:"login"`
	} `json:"user"`
	Labels []struct {
		Name string `json:"name"`
	} `json:"labels"`
}

type Options struct {
	State        string // "open", "closed", "all"
	AuthorOnly   bool
	ReviewOnly   bool
	IncludeDraft bool
}

func Fetch(owner, name string, opts Options) ([]PR, error) {
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

	type result struct {
		items []searchItem
		cat   string
		err   error
	}

	queries := []struct {
		q   string
		cat string
	}{}

	if !opts.ReviewOnly {
		q := buildQuery("is:pr", stateFilter, repo, "author:@me")
		queries = append(queries, struct {
			q   string
			cat string
		}{q, "author"})
	}
	if !opts.AuthorOnly {
		q := buildQuery("is:pr", stateFilter, repo, "review-requested:@me")
		queries = append(queries, struct {
			q   string
			cat string
		}{q, "review-requested"})
	}

	results := make([]result, len(queries))
	var wg sync.WaitGroup
	for i, qc := range queries {
		wg.Add(1)
		go func(idx int, query, cat string) {
			defer wg.Done()
			var resp searchResponse
			endpoint := "search/issues?per_page=100&q=" + url.QueryEscape(query)
			if err := client.Get(endpoint, &resp); err != nil {
				results[idx] = result{err: err, cat: cat}
				return
			}
			results[idx] = result{items: resp.Items, cat: cat}
		}(i, qc.q, qc.cat)
	}
	wg.Wait()

	seen := map[int]*PR{}
	var order []int

	for _, r := range results {
		if r.err != nil {
			return nil, fmt.Errorf("search failed (%s): %w", r.cat, r.err)
		}
		for _, item := range r.items {
			if !opts.IncludeDraft && item.Draft {
				continue
			}
			if existing, ok := seen[item.Number]; ok {
				existing.Categories = appendUniq(existing.Categories, r.cat)
				continue
			}
			pr := itemToPR(item, r.cat)
			seen[item.Number] = pr
			order = append(order, item.Number)
		}
	}

	prs := make([]PR, 0, len(order))
	for _, num := range order {
		prs = append(prs, *seen[num])
	}

	sort.Slice(prs, func(i, j int) bool {
		return prs[i].UpdatedAt.After(prs[j].UpdatedAt)
	})

	return prs, nil
}

func buildQuery(parts ...string) string {
	q := ""
	for _, p := range parts {
		if p == "" {
			continue
		}
		if q != "" {
			q += " "
		}
		q += p
	}
	return q
}

func itemToPR(item searchItem, cat string) *PR {
	t, _ := time.Parse(time.RFC3339, item.UpdatedAt)
	labels := make([]string, 0, len(item.Labels))
	for _, l := range item.Labels {
		labels = append(labels, l.Name)
	}
	return &PR{
		Number:     item.Number,
		Title:      item.Title,
		URL:        item.HTMLURL,
		Author:     item.User.Login,
		State:      item.State,
		IsDraft:    item.Draft,
		UpdatedAt:  t,
		Labels:     labels,
		Categories: []string{cat},
		Body:       item.Body,
	}
}

func appendUniq(slice []string, val string) []string {
	for _, s := range slice {
		if s == val {
			return slice
		}
	}
	return append(slice, val)
}
