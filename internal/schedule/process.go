package schedule

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

// EnsureDaemon spawns the daemon if it is not already running, or sends
// SIGHUP to trigger a registry reload if it is.
func EnsureDaemon() error {
	pid, err := readPID()
	if err == nil && pid > 0 {
		// Check liveness.
		if syscall.Kill(pid, 0) == nil {
			// Alive — request reload.
			return syscall.Kill(pid, syscall.SIGHUP)
		}
		// Stale PID file.
		pidPath, _ := PIDPath()
		os.Remove(pidPath)
	}
	return spawnDaemon()
}

func spawnDaemon() error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot resolve executable path: %w", err)
	}

	logPath, err := LogPath()
	if err != nil {
		return err
	}

	logFile, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("cannot open daemon log: %w", err)
	}
	defer logFile.Close()

	cmd := exec.Command(exe, "schedule", "_run-daemon")
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.Stdin = nil
	cmd.Env = os.Environ()
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start daemon: %w", err)
	}
	// Do not Wait — daemon runs independently.
	return nil
}

// StopDaemon sends SIGTERM to the daemon process.
func StopDaemon() error {
	pid, err := readPID()
	if err != nil {
		return fmt.Errorf("daemon not running (no PID file)")
	}
	if syscall.Kill(pid, 0) != nil {
		pidPath, _ := PIDPath()
		os.Remove(pidPath)
		return fmt.Errorf("daemon process %d not found (stale PID removed)", pid)
	}
	return syscall.Kill(pid, syscall.SIGTERM)
}

// DaemonPID returns (pid, running).
func DaemonPID() (int, bool) {
	pid, err := readPID()
	if err != nil || pid <= 0 {
		return 0, false
	}
	return pid, syscall.Kill(pid, 0) == nil
}

func readPID() (int, error) {
	path, err := PIDPath()
	if err != nil {
		return 0, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(string(data)))
}

func writePID(pid int) error {
	path, err := PIDPath()
	if err != nil {
		return err
	}
	return os.WriteFile(path, []byte(strconv.Itoa(pid)), 0644)
}

func removePID() {
	path, _ := PIDPath()
	os.Remove(path)
}
