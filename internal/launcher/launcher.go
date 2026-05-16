package launcher

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/Rchobbytech-Sols-Pvt-Ltd/skynetgcs/internal/config"
	"github.com/Rchobbytech-Sols-Pvt-Ltd/skynetgcs/internal/storage"
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

type childProcess struct {
	pid  int
	kill func() error
	wait func() error
}

type Manager struct {
	mu       sync.Mutex
	children map[string]*childProcess
}

func NewManager() *Manager {
	return &Manager{children: make(map[string]*childProcess)}
}

func installDir() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	dir := filepath.Dir(exe)

	// If running from Wails/Go default build output (build/bin),
	// go up to the project root to find sub-resources.
	// This ensures "skynetgcs/airunit" structure works in both dev and prod.
	if strings.HasSuffix(filepath.ToSlash(dir), "/build/bin") {
		return filepath.Dir(filepath.Dir(dir)), nil
	}

	return dir, nil
}

// MissingComponents returns the AssetPrefix of every component whose
// executable can't be located on disk. The search mirrors startOne:
// first <installDir>/<Subdir>/<Exe>, then <installDir>/<Exe>.
func MissingComponents() ([]string, error) {
	base, err := installDir()
	if err != nil {
		return nil, err
	}
	var missing []string
	for _, c := range config.Components {
		if _, err := os.Stat(filepath.Join(base, c.Subdir, c.Exe)); err == nil {
			continue
		}
		if _, err := os.Stat(filepath.Join(base, c.Exe)); err == nil {
			continue
		}
		missing = append(missing, c.AssetPrefix)
	}
	return missing, nil
}

func (m *Manager) StartAll() ([]ChildStatus, error) {
	base, err := installDir()
	if err != nil {
		log.Printf("[launcher] Failed to determine install directory: %v", err)
		return nil, err
	}
	log.Printf("[launcher] Starting all components. Base directory: %s", base)

	settings, err := storage.LoadSettings()
	if err != nil {
		log.Printf("[launcher] Failed to load settings, defaulting to hidden consoles: %v", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	results := make([]ChildStatus, 0, len(config.Components))
	for _, c := range config.Components {
		results = append(results, m.startOne(base, c, settings.ShowComponentConsoles))
	}
	return results, nil
}

func (m *Manager) startOne(base string, c config.Component, showConsole bool) ChildStatus {
	status := ChildStatus{Name: c.Subdir}

	if child, ok := m.children[c.Subdir]; ok {
		status.PID = child.pid
		status.Running = true
		return status
	}

	// Search strategy: 1. check the designated subdirectory, 2. fallback to root directory
	exePath := filepath.Join(base, c.Subdir, c.Exe)
	log.Printf("[launcher] [%s] Searching for executable at: %s", c.Subdir, exePath)
	if _, err := os.Stat(exePath); err != nil {
		// Fallback: Check if the executable is sitting directly in the root next to the launcher
		exePath = filepath.Join(base, c.Exe)
		log.Printf("[launcher] [%s] Not found in subdirectory, checking root fallback: %s", c.Subdir, exePath)
		if _, err := os.Stat(exePath); err != nil {
			log.Printf("[launcher] [%s] FAILED: Executable not found in subdirectory or root: %v", c.Subdir, err)
			status.Code = CodeNotInstalled
			return status
		}
		log.Printf("[launcher] [%s] Found in root directory fallback", c.Subdir)
	}

	// Keep Python-based components in UTF-8/unbuffered mode while launching them
	// like a normal hidden process, without redirecting stdout/stderr.
	log.Printf("[launcher] [%s] Setting Python environment (UTF-8, Unbuffered)", c.Subdir)
	env := append(os.Environ(),
		"PYTHONIOENCODING=utf-8",
		"PYTHONUTF8=1",
		"PYTHONUNBUFFERED=1",
	)

	child, err := startProcess(exePath, filepath.Dir(exePath), env, showConsole)
	if err != nil {
		log.Printf("[launcher] %s spawn failed: %v", c.Subdir, err)
		status.Code = CodeSpawnFailed
		status.Error = err.Error()
		return status
	}

	m.children[c.Subdir] = child
	status.PID = child.pid
	status.Running = true
	log.Printf("[launcher] %s started pid=%d", c.Subdir, child.pid)

	go func(name string, child *childProcess) {
		err := child.wait()

		m.mu.Lock()
		if cur, ok := m.children[name]; ok && cur == child {
			delete(m.children, name)
			log.Printf("[launcher] [%s] Process exited. Result: %v", name, err)
		}
		m.mu.Unlock()
	}(c.Subdir, child)

	return status
}

func (m *Manager) StopAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for name, child := range m.children {
		_ = child.kill()
		delete(m.children, name)
	}
}

func (m *Manager) Status() []ChildStatus {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]ChildStatus, 0, len(config.Components))
	for _, c := range config.Components {
		s := ChildStatus{Name: c.Subdir}
		if child, ok := m.children[c.Subdir]; ok {
			s.PID = child.pid
			s.Running = true
		}
		out = append(out, s)
	}
	return out
}
