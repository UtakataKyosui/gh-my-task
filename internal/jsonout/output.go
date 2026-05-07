package jsonout

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/UtakataKyosui/gh-my-task/internal/ghclient"
)

type Output struct {
	Repository string       `json:"repository"`
	User       string       `json:"user"`
	FetchedAt  time.Time    `json:"fetchedAt"`
	PRs        []ghclient.PR `json:"prs"`
}

func Print(owner, name, user string, prs []ghclient.PR) error {
	out := Output{
		Repository: fmt.Sprintf("%s/%s", owner, name),
		User:       user,
		FetchedAt:  time.Now().UTC(),
		PRs:        prs,
	}
	if out.PRs == nil {
		out.PRs = []ghclient.PR{}
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}
