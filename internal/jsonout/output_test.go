package jsonout

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/UtakataKyosui/gh-my-task/internal/ghclient"
)

var testPRs = []ghclient.PR{
	{Number: 1, Title: "fix bug", URL: "https://github.com/o/r/pull/1", Author: "alice", State: "open", UpdatedAt: time.Unix(0, 0).UTC()},
}

func TestPrintMatchesWriteFile(t *testing.T) {
	// Capture stdout output from Print.
	out := Build("owner", "repo", "user", testPRs, nil)
	var buf bytes.Buffer
	if err := Write(&buf, out); err != nil {
		t.Fatalf("Write: %v", err)
	}

	// Write to temp file and compare.
	dir := t.TempDir()
	path := filepath.Join(dir, "out", "current.json")
	// Use same Output so FetchedAt matches.
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	tmp := path + ".tmp"
	f, _ := os.Create(tmp)
	if err := Write(f, out); err != nil {
		f.Close()
		t.Fatalf("Write file: %v", err)
	}
	f.Close()
	os.Rename(tmp, path)

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !bytes.Equal(buf.Bytes(), got) {
		t.Errorf("Print and WriteFile produce different output:\nPrint:\n%s\nWriteFile:\n%s", buf.Bytes(), got)
	}
}

func TestWriteFileAtomicNoTmpOnSuccess(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "current.json")

	if err := WriteFile(path, "o", "r", "u", testPRs, nil); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if _, err := os.Stat(path + ".tmp"); !os.IsNotExist(err) {
		t.Errorf("tmp file should not exist after successful write")
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("output file should exist: %v", err)
	}
}

func TestWriteFileMkdirAll(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a", "b", "c", "current.json")
	if err := WriteFile(path, "o", "r", "u", nil, nil); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("output file should exist: %v", err)
	}
}
