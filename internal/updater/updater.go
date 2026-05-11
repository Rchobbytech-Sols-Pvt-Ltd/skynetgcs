package updater

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
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

	// Implementation Note: In a real scenario, you should call 
	// a 'StopAll' command here to ensure files are not locked
	// by running processes before extraction.

	exePath, err := os.Executable()
	if err != nil {
		return err
	}
	installDir := filepath.Dir(exePath)

	// If running from build/bin, ensure updates are applied to the 
	// project root instead of the build folder.
	if strings.HasSuffix(filepath.ToSlash(installDir), "/build/bin") {
		installDir = filepath.Dir(filepath.Dir(installDir))
	}

	tmpDir, err := os.MkdirTemp("", "skynetgcs-update-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	var missing []string
	for _, c := range config.Components {
		asset := findAsset(release.Assets, c.AssetPrefix)
		if asset == nil {
			missing = append(missing, c.AssetPrefix)
			continue
		}

		zipPath := filepath.Join(tmpDir, asset.Name)
		if err := download(asset.BrowserDownloadURL, zipPath); err != nil {
			return fmt.Errorf("download %s: %w", c.AssetPrefix, err)
		}

		dest := filepath.Join(installDir, c.Subdir)
		if err := Extract(zipPath, dest); err != nil {
			return fmt.Errorf("extract %s: %w", c.AssetPrefix, err)
		}
	}

	if len(missing) > 0 {
		return errors.New("missing release assets: " + strings.Join(missing, ", "))
	}

	refreshIconCache()
	return nil
}

// refreshIconCache notifies the Windows Shell that icons or associations have changed.
// This helps resolve the common issue where Windows shows a cached "old" icon.
func refreshIconCache() {
	if runtime.GOOS == "windows" {
		shell32 := syscall.NewLazyDLL("shell32.dll")
		proc := shell32.NewProc("SHChangeNotify")
		// SHCNE_ASSOCCHANGED = 0x08000000, SHCNF_IDLIST = 0
		_, _, _ = proc.Call(0x08000000, 0, 0, 0)
	}
}

// findAsset returns the first release asset whose filename starts with the
// given prefix (case-insensitive) and ends with ".zip". This lets release
// artifacts include arbitrary version suffixes — e.g. a prefix of "AirUnit"
// matches "AirUnit v1.2.2-alpha.zip", "airunit v1.3.0.zip", etc.
func findAsset(assets []ReleaseAsset, prefix string) *ReleaseAsset {
	p := strings.ToLower(prefix)
	for i, a := range assets {
		n := strings.ToLower(a.Name)
		if strings.HasPrefix(n, p) && strings.HasSuffix(n, ".zip") {
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
