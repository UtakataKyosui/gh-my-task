package jsonout

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/UtakataKyosui/gh-my-task/internal/ghclient"
)

type Output struct {
	Repository string           `json:"repository"`
	User       string           `json:"user"`
	FetchedAt  time.Time        `json:"fetchedAt"`
	PRs        []ghclient.PR    `json:"prs"`
	Issues     []ghclient.Issue `json:"issues,omitempty"`
}

func Build(owner, name, user string, prs []ghclient.PR, issues []ghclient.Issue) Output {
	out := Output{
		Repository: fmt.Sprintf("%s/%s", owner, name),
		User:       user,
		FetchedAt:  time.Now().UTC(),
		PRs:        prs,
		Issues:     issues,
	}
	if out.PRs == nil {
		out.PRs = []ghclient.PR{}
	}
	return out
}

func Write(w io.Writer, out Output) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func Print(owner, name, user string, prs []ghclient.PR, issues []ghclient.Issue) error {
	return Write(os.Stdout, Build(owner, name, user, prs, issues))
}

func WriteFile(path, owner, name, user string, prs []ghclient.PR, issues []ghclient.Issue) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if err := Write(f, Build(owner, name, user, prs, issues)); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	f.Close()
	return os.Rename(tmp, path)
}
