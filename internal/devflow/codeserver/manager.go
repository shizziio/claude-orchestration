package codeserver

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"sync"
	"syscall"
)

// Manager tracks running code-server instances by port.
type Manager struct {
	mu   sync.Mutex
	pids map[int]int // port → PID
}

// NewManager creates a new code-server process manager.
func NewManager() *Manager {
	return &Manager{pids: make(map[int]int)}
}

// Start launches code-server bound to the given port and serving projectDir.
func (m *Manager) Start(projectDir string, port int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if already running on this port
	if pid, ok := m.pids[port]; ok {
		if processAlive(pid) {
			return fmt.Errorf("code-server already running on port %d (pid %d)", port, pid)
		}
		// Stale entry — clean up
		delete(m.pids, port)
	}

	cmd := exec.Command("code-server",
		"--bind-addr", fmt.Sprintf("0.0.0.0:%d", port),
		"--auth", "none",
		projectDir,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	// Detach from parent process group so it survives handler return
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start code-server: %w", err)
	}

	m.pids[port] = cmd.Process.Pid
	slog.Info("devflow.codeserver.started", "port", port, "pid", cmd.Process.Pid, "dir", projectDir)

	// Release the process so it doesn't become a zombie
	go cmd.Wait()

	return nil
}

// Stop kills the code-server process on the given port.
func (m *Manager) Stop(port int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	pid, ok := m.pids[port]
	if !ok {
		return fmt.Errorf("no code-server running on port %d", port)
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		delete(m.pids, port)
		return fmt.Errorf("process %d not found: %w", pid, err)
	}

	if err := proc.Signal(syscall.SIGTERM); err != nil {
		// Try SIGKILL as fallback
		_ = proc.Kill()
	}

	delete(m.pids, port)
	slog.Info("devflow.codeserver.stopped", "port", port, "pid", pid)
	return nil
}

// IsRunning checks if a code-server process is alive on the given port.
func (m *Manager) IsRunning(port int) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	pid, ok := m.pids[port]
	if !ok {
		return false
	}
	if !processAlive(pid) {
		delete(m.pids, port)
		return false
	}
	return true
}

// NextPort returns the next available port starting from basePort.
func (m *Manager) NextPort(basePort int) int {
	m.mu.Lock()
	defer m.mu.Unlock()

	port := basePort
	for {
		if _, ok := m.pids[port]; !ok {
			return port
		}
		port++
	}
}

// processAlive checks if a process with the given PID is still running.
func processAlive(pid int) bool {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// Signal 0 checks existence without actually sending a signal
	return proc.Signal(syscall.Signal(0)) == nil
}
