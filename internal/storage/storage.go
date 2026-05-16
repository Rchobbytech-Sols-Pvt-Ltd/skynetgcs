package storage

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/Rchobbytech-Sols-Pvt-Ltd/skynetgcs/internal/config"
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

type ComponentRecord struct {
	Version      string    `json:"version"`
	AssetName    string    `json:"asset_name,omitempty"`
	DownloadedAt time.Time `json:"downloaded_at"`
}

type DownloadRecord struct {
	Components map[string]ComponentRecord `json:"components"`
}

func downloadRecordPath() (string, error) {
	dir, err := dataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "downloaded.json"), nil
}

func LoadDownloadRecord() (*DownloadRecord, error) {
	path, err := downloadRecordPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &DownloadRecord{Components: map[string]ComponentRecord{}}, nil
		}
		return nil, err
	}
	var r DownloadRecord
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, err
	}
	if r.Components == nil {
		r.Components = map[string]ComponentRecord{}
	}
	return &r, nil
}

func saveDownloadRecord(r *DownloadRecord) error {
	path, err := downloadRecordPath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// SaveComponentDownload records that the named component was successfully
// downloaded from the given release tag. The key should be the component's
// AssetPrefix as declared in config.Components.
func SaveComponentDownload(key, version, assetName string) error {
	r, err := LoadDownloadRecord()
	if err != nil {
		return err
	}
	r.Components[key] = ComponentRecord{
		Version:      version,
		AssetName:    assetName,
		DownloadedAt: time.Now().UTC(),
	}
	return saveDownloadRecord(r)
}

// ComponentVersion returns the recorded version of the named component, or
// an empty string if none has been recorded.
func ComponentVersion(key string) string {
	r, err := LoadDownloadRecord()
	if err != nil || r == nil {
		return ""
	}
	return r.Components[key].Version
}

type Settings struct {
	ShowComponentConsoles bool `json:"show_component_consoles"`
}

func settingsPath() (string, error) {
	dir, err := dataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "settings.json"), nil
}

func LoadSettings() (Settings, error) {
	path, err := settingsPath()
	if err != nil {
		return Settings{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Settings{}, nil
		}
		return Settings{}, err
	}
	var s Settings
	if err := json.Unmarshal(data, &s); err != nil {
		return Settings{}, err
	}
	return s, nil
}

func SaveSettings(s Settings) error {
	path, err := settingsPath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
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
