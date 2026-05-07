package schedule

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"syscall"
	"time"
)

type Registry struct {
	Version   int        `json:"version"`
	Schedules []Schedule `json:"schedules"`
	mu        sync.Mutex
}

func LoadRegistry() (*Registry, error) {
	path, err := RegistryPath()
	if err != nil {
		return &Registry{Version: 1, Schedules: []Schedule{}}, nil
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &Registry{Version: 1, Schedules: []Schedule{}}, nil
	}
	if err != nil {
		return nil, err
	}
	var r Registry
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, err
	}
	if r.Schedules == nil {
		r.Schedules = []Schedule{}
	}
	return &r, nil
}

func (r *Registry) Save() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.saveLocked()
}

func (r *Registry) saveLocked() error {
	path, err := RegistryPath()
	if err != nil {
		return err
	}
	lockPath, err := registryLockPath()
	if err != nil {
		return err
	}

	lf, err := os.OpenFile(lockPath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer lf.Close()
	if err := syscall.Flock(int(lf.Fd()), syscall.LOCK_EX); err != nil {
		return err
	}
	defer syscall.Flock(int(lf.Fd()), syscall.LOCK_UN) //nolint:errcheck

	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func scheduleID(repoPath string) string {
	h := sha1.Sum([]byte(repoPath))
	return fmt.Sprintf("%x", h[:4])
}

func (r *Registry) Add(s Schedule) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s.ID = scheduleID(s.RepoPath)
	for i, existing := range r.Schedules {
		if existing.RepoPath == s.RepoPath {
			r.Schedules[i] = s
			return
		}
	}
	r.Schedules = append(r.Schedules, s)
}

func (r *Registry) Remove(repoPath string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, s := range r.Schedules {
		if s.RepoPath == repoPath {
			r.Schedules = append(r.Schedules[:i], r.Schedules[i+1:]...)
			return true
		}
	}
	return false
}

func (r *Registry) Find(repoPath string) (Schedule, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, s := range r.Schedules {
		if s.RepoPath == repoPath {
			return s, true
		}
	}
	return Schedule{}, false
}

func (r *Registry) Snapshot() []Schedule {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]Schedule, len(r.Schedules))
	copy(out, r.Schedules)
	return out
}

// ApplyStatus reloads registry from disk, updates the given schedule's run
// metadata, and saves. Best-effort — errors are silently dropped since status
// updates must not crash the daemon.
func ApplyStatus(id string, lastRun, nextRun time.Time, errStr string) {
	r, err := LoadRegistry()
	if err != nil {
		return
	}
	r.mu.Lock()
	for i, s := range r.Schedules {
		if s.ID == id {
			r.Schedules[i].LastRun = lastRun
			r.Schedules[i].NextRun = nextRun
			r.Schedules[i].LastError = errStr
			break
		}
	}
	r.saveLocked() //nolint:errcheck
	r.mu.Unlock()
}
