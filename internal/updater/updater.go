package updater

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jhakrishan20/skynetgcs/internal/config"
)

type ReleaseAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

type ReleaseInfo struct {
	TagName     string         `json:"tag_name"`
	Name        string         `json:"name"`
	Body        string         `json:"body"`
	PublishedAt time.Time      `json:"published_at"`
	Assets      []ReleaseAsset `json:"assets"`
}

func githubLatestURL() string {
	return fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest",
		config.GitHubOwner, config.GitHubRepo)
}

func CheckLatest() (*ReleaseInfo, error) {
	req, err := http.NewRequest(http.MethodGet, githubLatestURL(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github releases: status %d", resp.StatusCode)
	}

	var info ReleaseInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, err
	}
	return &info, nil
}

func IsNewer(release *ReleaseInfo) bool {
	if release == nil {
		return false
	}
	current := strings.TrimPrefix(config.AppVersion, "v")
	latest := strings.TrimPrefix(release.TagName, "v")
	return latest != "" && latest != current
}

func DownloadAndApply(release *ReleaseInfo) error {
	if release == nil {
		return fmt.Errorf("nil release")
	}

	exePath, err := os.Executable()
	if err != nil {
		return err
	}
	installDir := filepath.Dir(exePath)

	tmpDir, err := os.MkdirTemp("", "skynetgcs-update-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	var missing []string
	for _, c := range config.Components {
		asset := findAsset(release.Assets, c.AssetName)
		if asset == nil {
			missing = append(missing, c.AssetName)
			continue
		}

		zipPath := filepath.Join(tmpDir, asset.Name)
		if err := download(asset.BrowserDownloadURL, zipPath); err != nil {
			return fmt.Errorf("download %s: %w", c.AssetName, err)
		}

		dest := filepath.Join(installDir, c.Subdir)
		if err := Extract(zipPath, dest); err != nil {
			return fmt.Errorf("extract %s: %w", c.AssetName, err)
		}
	}

	if len(missing) > 0 {
		return errors.New("missing release assets: " + strings.Join(missing, ", "))
	}
	return nil
}

func findAsset(assets []ReleaseAsset, name string) *ReleaseAsset {
	target := strings.ToLower(name)
	for i, a := range assets {
		if strings.ToLower(a.Name) == target {
			return &assets[i]
		}
	}
	return nil
}

func download(url, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download: status %d", resp.StatusCode)
	}

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}
