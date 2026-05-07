package schedule

import (
	"os"
	"path/filepath"
)

func configDir() (string, error) {
	if override := os.Getenv("GH_MY_TASK_CONFIG_DIR"); override != "" {
		if err := os.MkdirAll(override, 0755); err != nil {
			return "", err
		}
		return override, nil
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, "gh-my-task")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return dir, nil
}

func RegistryPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "schedules.json"), nil
}

func registryLockPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "schedules.json.lock"), nil
}

func PIDPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "daemon.pid"), nil
}

func LogPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "daemon.log"), nil
}

// RepoOutputPath returns <repoPath>/.git/my-tasks/current.json.
func RepoOutputPath(repoPath string) string {
	return filepath.Join(repoPath, ".git", "my-tasks", "current.json")
}
