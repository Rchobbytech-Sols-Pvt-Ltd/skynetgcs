package launcher

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/jhakrishan20/skynetgcs/internal/config"
)

// Stable codes the frontend uses to render user-facing messages.
const (
	CodeNotInstalled = "not_installed"
	CodeSpawnFailed  = "spawn_failed"
)

type ChildStatus struct {
	Name    string `json:"name"`
	PID     int    `json:"pid"`
	Running bool   `json:"running"`
	Code    string `json:"code,omitempty"`
	Error   string `json:"error,omitempty"`
}

type Manager struct {
	mu       sync.Mutex
	children map[string]*exec.Cmd
}

func NewManager() *Manager {
	return &Manager{children: make(map[string]*exec.Cmd)}
}

func installDir() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Dir(exe), nil
}

func (m *Manager) StartAll() ([]ChildStatus, error) {
	base, err := installDir()
	if err != nil {
		return nil, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	results := make([]ChildStatus, 0, len(config.Components))
	for _, c := range config.Components {
		results = append(results, m.startOne(base, c))
	}
	return results, nil
}

func (m *Manager) startOne(base string, c config.Component) ChildStatus {
	status := ChildStatus{Name: c.Subdir}

	if cmd, ok := m.children[c.Subdir]; ok && cmd.Process != nil && cmd.ProcessState == nil {
		status.PID = cmd.Process.Pid
		status.Running = true
		return status
	}

	exePath := filepath.Join(base, c.Subdir, c.Exe)
	if _, err := os.Stat(exePath); err != nil {
		log.Printf("[launcher] %s not installed: %s (%v)", c.Subdir, exePath, err)
		status.Code = CodeNotInstalled
		return status
	}

	cmd := exec.Command(exePath)
	cmd.Dir = filepath.Dir(exePath)
	hideConsole(cmd)

	if err := cmd.Start(); err != nil {
		log.Printf("[launcher] %s spawn failed: %v", c.Subdir, err)
		status.Code = CodeSpawnFailed
		status.Error = err.Error()
		return status
	}

	m.children[c.Subdir] = cmd
	status.PID = cmd.Process.Pid
	status.Running = true
	log.Printf("[launcher] %s started pid=%d", c.Subdir, cmd.Process.Pid)

	go func(name string, cmd *exec.Cmd) {
		_ = cmd.Wait()
		m.mu.Lock()
		if cur, ok := m.children[name]; ok && cur == cmd {
			delete(m.children, name)
			log.Printf("[launcher] %s exited", name)
		}
		m.mu.Unlock()
	}(c.Subdir, cmd)

	return status
}

func (m *Manager) StopAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for name, cmd := range m.children {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		delete(m.children, name)
	}
}

func (m *Manager) Status() []ChildStatus {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]ChildStatus, 0, len(config.Components))
	for _, c := range config.Components {
		s := ChildStatus{Name: c.Subdir}
		if cmd, ok := m.children[c.Subdir]; ok && cmd.Process != nil {
			s.PID = cmd.Process.Pid
			s.Running = true
		}
		out = append(out, s)
	}
	return out
}
