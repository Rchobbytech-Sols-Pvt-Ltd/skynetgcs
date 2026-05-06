package storage

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/jhakrishan20/skynetgcs/internal/config"
)

type Activation struct {
	Key         string    `json:"key"`
	ActivatedAt time.Time `json:"activated_at"`
}

func dataDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, config.AppName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

func activationPath() (string, error) {
	dir, err := dataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "activation.json"), nil
}

func SaveActivation(key string) error {
	path, err := activationPath()
	if err != nil {
		return err
	}
	a := Activation{Key: key, ActivatedAt: time.Now().UTC()}
	data, err := json.MarshalIndent(a, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func LoadActivation() (*Activation, error) {
	path, err := activationPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var a Activation
	if err := json.Unmarshal(data, &a); err != nil {
		return nil, err
	}
	return &a, nil
}

func IsActivated() bool {
	a, err := LoadActivation()
	if err != nil || a == nil {
		return false
	}
	return a.Key != ""
}
