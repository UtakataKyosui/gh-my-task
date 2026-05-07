package ghclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"sync"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
)

type PR struct {
	Number         int             `json:"number"`
	Title          string          `json:"title"`
	URL            string          `json:"url"`
	Author         string          `json:"author"`
	State          string          `json:"state"`
	IsDraft        bool            `json:"isDraft"`
	UpdatedAt      time.Time       `json:"updatedAt"`
	Labels         []string        `json:"labels"`
	Categories     []string        `json:"categories"`
	Body           string          `json:"body,omitempty"`
	HeadRef        string          `json:"headRef,omitempty"`
	Reviews        []Review        `json:"reviews,omitempty"`
	ReviewComments []ReviewComment `json:"reviewComments,omitempty"`
}

type Review struct {
	Reviewer string `json:"reviewer"`
	State    string `json:"state"` // APPROVED, CHANGES_REQUESTED, COMMENTED
}

type ReviewComment struct {
	Reviewer string `json:"reviewer"`
	Path     string `json:"path"`
	Line     int    `json:"line"`
	Body     string `json:"body"`
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
	WithReviews  bool // fetch review status for each PR (slower)
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

	if opts.WithReviews {
		client2, err2 := api.DefaultRESTClient()
		if err2 != nil {
			return prs, nil // reviews are best-effort
		}
		var rwg sync.WaitGroup
		for i := range prs {
			rwg.Add(1)
			go func(idx int) {
				defer rwg.Done()
				revs, _ := fetchReviewsWithClient(client2, owner, name, prs[idx].Number)
				prs[idx].Reviews = revs
			}(i)
		}
		rwg.Wait()
	}

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

func FetchReviews(owner, name string, number int) ([]Review, error) {
	client, err := api.DefaultRESTClient()
	if err != nil {
		return nil, err
	}
	return fetchReviewsWithClient(client, owner, name, number)
}

func fetchReviewsWithClient(client *api.RESTClient, owner, name string, number int) ([]Review, error) {
	var items []struct {
		User  struct{ Login string `json:"login"` } `json:"user"`
		State string                                `json:"state"`
	}
	path := fmt.Sprintf("repos/%s/%s/pulls/%d/reviews?per_page=100", owner, name, number)
	if err := client.Get(path, &items); err != nil {
		return nil, err
	}
	// keep latest state per reviewer
	latest := map[string]string{}
	for _, item := range items {
		if item.State == "PENDING" {
			continue
		}
		latest[item.User.Login] = item.State
	}
	revs := make([]Review, 0, len(latest))
	for reviewer, state := range latest {
		revs = append(revs, Review{Reviewer: reviewer, State: state})
	}
	sort.Slice(revs, func(i, j int) bool { return revs[i].Reviewer < revs[j].Reviewer })
	return revs, nil
}

func FetchReviewComments(owner, name string, number int) ([]ReviewComment, error) {
	client, err := api.DefaultRESTClient()
	if err != nil {
		return nil, err
	}
	var items []struct {
		User struct{ Login string `json:"login"` } `json:"user"`
		Path string                                `json:"path"`
		Line int                                   `json:"line"`
		Body string                                `json:"body"`
	}
	path := fmt.Sprintf("repos/%s/%s/pulls/%d/comments?per_page=100", owner, name, number)
	if err := client.Get(path, &items); err != nil {
		return nil, err
	}
	comments := make([]ReviewComment, 0, len(items))
	for _, item := range items {
		comments = append(comments, ReviewComment{
			Reviewer: item.User.Login,
			Path:     item.Path,
			Line:     item.Line,
			Body:     item.Body,
		})
	}
	return comments, nil
}

func FetchOne(owner, name string, number int) (PR, error) {
	client, err := api.DefaultRESTClient()
	if err != nil {
		return PR{}, fmt.Errorf("failed to create API client: %w", err)
	}
	var resp struct {
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
		Head struct {
			Ref string `json:"ref"`
		} `json:"head"`
	}
	path := fmt.Sprintf("repos/%s/%s/pulls/%d", owner, name, number)
	if err := client.Get(path, &resp); err != nil {
		return PR{}, err
	}
	t, _ := time.Parse(time.RFC3339, resp.UpdatedAt)
	pr := PR{
		Number:    resp.Number,
		Title:     resp.Title,
		URL:       resp.HTMLURL,
		Author:    resp.User.Login,
		State:     resp.State,
		IsDraft:   resp.Draft,
		UpdatedAt: t,
		Body:      resp.Body,
		HeadRef:   resp.Head.Ref,
	}
	// fetch reviews and review comments concurrently
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		pr.Reviews, _ = FetchReviews(owner, name, number)
	}()
	go func() {
		defer wg.Done()
		pr.ReviewComments, _ = FetchReviewComments(owner, name, number)
	}()
	wg.Wait()
	return pr, nil
}

func CurrentUser() string {
	client, err := api.DefaultRESTClient()
	if err != nil {
		return ""
	}
	var resp struct {
		Login string `json:"login"`
	}
	if err := client.Get("user", &resp); err != nil {
		return ""
	}
	return resp.Login
}

func ClosePR(owner, name string, number int) error {
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
		fmt.Sprintf("repos/%s/%s/pulls/%d", owner, name, number),
		bytes.NewReader(bodyBytes),
		&resp,
	)
}
