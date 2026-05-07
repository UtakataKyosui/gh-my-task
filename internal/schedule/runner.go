package schedule

import (
	"context"
	"log"
	"sync"
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
		var (
			prs    []ghclient.PR
			issues []ghclient.Issue
			prErr  error
		)
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			prs, prErr = ghclient.Fetch(s.Owner, s.Name, s.Filter.ToOptions())
		}()
		if s.Filter.WithIssues {
			wg.Add(1)
			go func() {
				defer wg.Done()
				var e error
				issues, e = ghclient.FetchIssues(s.Owner, s.Name, s.Filter.ToIssueOptions())
				if e != nil {
					log.Printf("[runner %s] fetch issues error: %v", s.ID, e)
				}
			}()
		}
		wg.Wait()
		if prErr != nil {
			log.Printf("[runner %s] fetch error: %v", s.ID, prErr)
			select {
			case statusCh <- StatusUpdate{ID: s.ID, LastRun: now, NextRun: nextRun, Err: prErr.Error()}:
			default:
			}
			return
		}
		path := RepoOutputPath(s.RepoPath)
		if err := jsonout.WriteFile(path, s.Owner, s.Name, user, prs, issues); err != nil {
			log.Printf("[runner %s] write error: %v", s.ID, err)
			select {
			case statusCh <- StatusUpdate{ID: s.ID, LastRun: now, NextRun: nextRun, Err: err.Error()}:
			default:
			}
			return
		}
		log.Printf("[runner %s] wrote %d PRs, %d issues to %s", s.ID, len(prs), len(issues), path)
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
