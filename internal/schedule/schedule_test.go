package schedule

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/UtakataKyosui/gh-my-task/internal/ghclient"
)

func TestFilterToOptions(t *testing.T) {
	f := Filter{
		State:        "closed",
		AuthorOnly:   true,
		ReviewOnly:   false,
		IncludeDraft: false,
		WithReviews:  true,
	}
	opts := f.ToOptions()
	want := ghclient.Options{
		State:        "closed",
		AuthorOnly:   true,
		ReviewOnly:   false,
		IncludeDraft: false,
		WithReviews:  true,
	}
	if opts != want {
		t.Errorf("ToOptions() = %+v, want %+v", opts, want)
	}
}

func TestDurationMarshalUnmarshal(t *testing.T) {
	original := duration{5 * time.Minute}
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(data) != `"5m0s"` {
		t.Errorf("expected %q, got %s", "5m0s", data)
	}
	var got duration
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Duration != original.Duration {
		t.Errorf("round-trip mismatch: %v vs %v", got, original)
	}
}

func TestScheduleID(t *testing.T) {
	id1 := scheduleID("/path/to/repo")
	id2 := scheduleID("/path/to/repo")
	id3 := scheduleID("/other/repo")
	if id1 != id2 {
		t.Errorf("same path should produce same ID")
	}
	if id1 == id3 {
		t.Errorf("different paths should produce different IDs (collision unlikely)")
	}
}
