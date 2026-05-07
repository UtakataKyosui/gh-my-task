package schedule

import (
	"sync"
	"testing"
	"time"
)

func TestRegistryRoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GH_MY_TASK_CONFIG_DIR", dir) // Linux-style; macOS uses UserConfigDir but we override via env

	s := Schedule{
		RepoPath:  "/tmp/repo",
		Owner:     "owner",
		Name:      "name",
		Interval:  duration{5 * time.Minute},
		Filter:    Filter{State: "open", IncludeDraft: true},
		CreatedAt: time.Now().UTC().Truncate(time.Second),
	}

	reg := &Registry{Version: 1, Schedules: []Schedule{}}
	reg.Add(s)
	if err := reg.Save(); err != nil {
		t.Fatalf("save: %v", err)
	}

	reg2, err := LoadRegistry()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(reg2.Schedules) != 1 {
		t.Fatalf("expected 1 schedule, got %d", len(reg2.Schedules))
	}
	got := reg2.Schedules[0]
	if got.Owner != s.Owner || got.Name != s.Name {
		t.Errorf("owner/name mismatch: %s/%s", got.Owner, got.Name)
	}
	if got.Interval.Duration != s.Interval.Duration {
		t.Errorf("interval mismatch: %v vs %v", got.Interval, s.Interval)
	}
	if got.Filter.State != s.Filter.State {
		t.Errorf("filter.state mismatch: %q vs %q", got.Filter.State, s.Filter.State)
	}
}

func TestRegistryAddRemoveFind(t *testing.T) {
	reg := &Registry{Version: 1, Schedules: []Schedule{}}

	reg.Add(Schedule{RepoPath: "/a", Owner: "o", Name: "a", Interval: duration{time.Minute}})
	reg.Add(Schedule{RepoPath: "/b", Owner: "o", Name: "b", Interval: duration{time.Minute}})

	if s, ok := reg.Find("/a"); !ok || s.Name != "a" {
		t.Errorf("find /a failed")
	}
	if !reg.Remove("/a") {
		t.Errorf("remove /a failed")
	}
	if _, ok := reg.Find("/a"); ok {
		t.Errorf("found /a after removal")
	}
	if len(reg.Schedules) != 1 {
		t.Errorf("expected 1 schedule after removal, got %d", len(reg.Schedules))
	}
}

func TestRegistryAddUpsert(t *testing.T) {
	reg := &Registry{Version: 1, Schedules: []Schedule{}}
	reg.Add(Schedule{RepoPath: "/r", Owner: "o", Name: "old", Interval: duration{time.Minute}})
	reg.Add(Schedule{RepoPath: "/r", Owner: "o", Name: "new", Interval: duration{2 * time.Minute}})
	if len(reg.Schedules) != 1 {
		t.Fatalf("expected 1 (upsert), got %d", len(reg.Schedules))
	}
	if reg.Schedules[0].Name != "new" {
		t.Errorf("upsert did not update name")
	}
}

func TestRegistryConcurrentSave(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GH_MY_TASK_CONFIG_DIR", dir)

	reg := &Registry{Version: 1, Schedules: []Schedule{
		{RepoPath: "/x", Owner: "o", Name: "x", Interval: duration{time.Minute}},
	}}

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			reg.Save() //nolint:errcheck
		}()
	}
	wg.Wait()

	// Verify file is valid JSON after concurrent saves.
	reg2, err := LoadRegistry()
	if err != nil {
		t.Fatalf("load after concurrent save: %v", err)
	}
	if len(reg2.Schedules) != 1 {
		t.Errorf("expected 1 schedule, got %d", len(reg2.Schedules))
	}
}

func TestRegistryLoadMissing(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GH_MY_TASK_CONFIG_DIR", dir)

	reg, err := LoadRegistry()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reg == nil || len(reg.Schedules) != 0 {
		t.Errorf("expected empty registry, got %+v", reg)
	}
}
