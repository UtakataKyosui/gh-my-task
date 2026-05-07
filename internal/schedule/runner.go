package schedule

import (
	"context"
	"log"
	"time"

	"github.com/UtakataKyosui/gh-my-task/internal/ghclient"
	"github.com/UtakataKyosui/gh-my-task/internal/jsonout"
)

type StatusUpdate struct {
	ID      string
	LastRun time.Time
	NextRun time.Time
	Err     string
}

func Run(ctx context.Context, s Schedule, statusCh chan<- StatusUpdate) {
	user := ghclient.CurrentUser()

	tick := func() (nextRun time.Time) {
		now := time.Now()
		nextRun = now.Add(s.Interval.Duration)
		prs, err := ghclient.Fetch(s.Owner, s.Name, s.Filter.ToOptions())
		if err != nil {
			log.Printf("[runner %s] fetch error: %v", s.ID, err)
			select {
			case statusCh <- StatusUpdate{ID: s.ID, LastRun: now, NextRun: nextRun, Err: err.Error()}:
			default:
			}
			return
		}
		path := RepoOutputPath(s.RepoPath)
		if err := jsonout.WriteFile(path, s.Owner, s.Name, user, prs); err != nil {
			log.Printf("[runner %s] write error: %v", s.ID, err)
			select {
			case statusCh <- StatusUpdate{ID: s.ID, LastRun: now, NextRun: nextRun, Err: err.Error()}:
			default:
			}
			return
		}
		log.Printf("[runner %s] wrote %d PRs to %s", s.ID, len(prs), path)
		select {
		case statusCh <- StatusUpdate{ID: s.ID, LastRun: now, NextRun: nextRun}:
		default:
		}
		return
	}

	next := tick()

	t := time.NewTimer(time.Until(next))
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			next = tick()
			t.Reset(time.Until(next))
		}
	}
}
