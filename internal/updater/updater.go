package updater

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/Rchobbytech-Sols-Pvt-Ltd/skynetgcs/internal/config"
	"github.com/Rchobbytech-Sols-Pvt-Ltd/skynetgcs/internal/storage"
)

// componentInstalledVersion returns the recorded version of the given
// component. If nothing has been recorded yet (e.g. this launcher build
// shipped before per-component tracking, or the user has never run an
// update), we fall back to the launcher's compiled-in AppVersion — the
// build is assumed to match its release tag's components.
func componentInstalledVersion(prefix string) string {
	if v := storage.ComponentVersion(prefix); v != "" {
		return v
	}
	return config.AppVersion
}

// releaseHasUpdates reports whether any component declared in
// config.Components is both present in the release AND at a version
// different from what is recorded on disk. Components missing from the
// release are ignored (we don't downgrade or flag them).
func releaseHasUpdates(release *ReleaseInfo) bool {
	if release == nil {
		return false
	}
	latestTag := strings.TrimPrefix(release.TagName, "v")
	if latestTag == "" {
		return false
	}
	for _, c := range config.Components {
		asset := findAsset(release.Assets, c.AssetPrefix)
		if asset == nil {
			continue
		}
		installed := strings.TrimPrefix(componentInstalledVersion(c.AssetPrefix), "v")
		if installed != latestTag {
			return true
		}
	}
	return false
}

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

type UpdateCheckResult struct {
	UpdateAvailable bool         `json:"update_available"`
	CurrentVersion  string       `json:"current_version"`
	LatestVersion   string       `json:"latest_version"`
	Release         *ReleaseInfo `json:"release,omitempty"`
}

type ProgressFunc func(downloadedBytes, totalBytes int64)
type byteProgressFunc func(bytesRead int64)

type componentAsset struct {
	component config.Component
	asset     *ReleaseAsset
}

func githubLatestURL() string {
	return fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest",
		config.GitHubOwner, config.GitHubRepo)
}

func CheckForUpdate() (*UpdateCheckResult, error) {
	log.Printf("[updater] Checking for updates. Launcher version: %s", config.AppVersion)
	release, err := CheckLatest()
	if err != nil {
		log.Printf("[updater] Check for updates failed: %v", err)
		return nil, err
	}

	log.Printf("[updater] Latest release found: %s", release.TagName)
	for _, c := range config.Components {
		recorded := storage.ComponentVersion(c.AssetPrefix)
		effective := componentInstalledVersion(c.AssetPrefix)
		source := "recorded"
		if recorded == "" {
			source = "fallback=AppVersion"
		}
		present := findAsset(release.Assets, c.AssetPrefix) != nil
		log.Printf("[updater]   component %s: installed=%s (%s), in-release=%t", c.AssetPrefix, effective, source, present)
	}

	return &UpdateCheckResult{
		UpdateAvailable: releaseHasUpdates(release),
		CurrentVersion:  config.AppVersion,
		LatestVersion:   release.TagName,
		Release:         release,
	}, nil
}

func CheckLatest() (*ReleaseInfo, error) {
	url := githubLatestURL()
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", config.AppName)

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("github releases %s: status %d: %s", url, resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var info ReleaseInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, err
	}
	return &info, nil
}

// IsNewer is kept for callers that want the old single-version check.
// Internally, update detection now goes through releaseHasUpdates so that
// partial releases (missing one of the components) don't cause false
// positives on the dashboard.
func IsNewer(release *ReleaseInfo) bool {
	return releaseHasUpdates(release)
}

func DownloadAndApply(release *ReleaseInfo) error {
	return DownloadAndApplyContext(context.Background(), release)
}

func DownloadAndApplyContext(ctx context.Context, release *ReleaseInfo, progressFns ...ProgressFunc) error {
	if release == nil {
		return fmt.Errorf("nil release")
	}
	log.Printf("[updater] Starting update to version: %s", release.TagName)

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
	log.Printf("[updater] Using temporary directory for update: %s", tmpDir)
	defer os.RemoveAll(tmpDir)

	// Build the download list from whatever components are present in the
	// release. Components missing from the release are skipped — this is a
	// partial-release-friendly flow: an HCI-only release should update HCI
	// without touching AirUnit. Components that already match the release
	// tag are also skipped so we don't re-download identical assets.
	latestTag := release.TagName
	var downloads []componentAsset
	var totalBytes int64
	var skippedUpToDate []string
	var skippedAbsent []string
	for _, c := range config.Components {
		asset := findAsset(release.Assets, c.AssetPrefix)
		if asset == nil {
			skippedAbsent = append(skippedAbsent, c.AssetPrefix)
			continue
		}
		installed := strings.TrimPrefix(componentInstalledVersion(c.AssetPrefix), "v")
		if installed != "" && installed == strings.TrimPrefix(latestTag, "v") {
			skippedUpToDate = append(skippedUpToDate, c.AssetPrefix)
			continue
		}
		downloads = append(downloads, componentAsset{component: c, asset: asset})
		totalBytes += asset.Size
	}

	if len(skippedAbsent) > 0 {
		log.Printf("[updater] Release %s does not contain: %s — leaving installed versions untouched", latestTag, strings.Join(skippedAbsent, ", "))
	}
	if len(skippedUpToDate) > 0 {
		log.Printf("[updater] Already at %s for: %s", latestTag, strings.Join(skippedUpToDate, ", "))
	}
	if len(downloads) == 0 {
		log.Printf("[updater] Nothing to download for release %s", latestTag)
		return nil
	}

	var downloadedBytes int64
	reportProgress(progressFns, downloadedBytes, totalBytes)

	for _, item := range downloads {
		if err := ctx.Err(); err != nil {
			return err
		}

		zipPath := filepath.Join(tmpDir, item.asset.Name)
		log.Printf("[updater] Downloading %s from %s", item.asset.Name, item.asset.BrowserDownloadURL)
		if err := download(ctx, item.asset.BrowserDownloadURL, zipPath, func(n int64) {
			downloadedBytes += n
			reportProgress(progressFns, downloadedBytes, totalBytes)
		}); err != nil {
			return fmt.Errorf("download %s: %w", item.component.AssetPrefix, err)
		}

		if err := ctx.Err(); err != nil {
			return err
		}

		dest := filepath.Join(installDir, item.component.Subdir)
		log.Printf("[updater] Extracting %s to %s", item.asset.Name, dest)
		if err := ExtractContext(ctx, zipPath, dest); err != nil {
			return fmt.Errorf("extract %s: %w", item.component.AssetPrefix, err)
		}

		if err := storage.SaveComponentDownload(item.component.AssetPrefix, release.TagName, item.asset.Name); err != nil {
			log.Printf("[updater] Warning: failed to record %s @ %s: %v", item.component.AssetPrefix, release.TagName, err)
		} else {
			log.Printf("[updater] Recorded %s @ %s", item.component.AssetPrefix, release.TagName)
		}
	}

	refreshIconCache()
	return nil
}

func reportProgress(progressFns []ProgressFunc, downloadedBytes, totalBytes int64) {
	for _, fn := range progressFns {
		if fn != nil {
			fn(downloadedBytes, totalBytes)
		}
	}
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

func download(ctx context.Context, url, dest string, progress byteProgressFunc) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", config.AppName)

	resp, err := http.DefaultClient.Do(req)
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

	buf := make([]byte, 64*1024)
	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, err := out.Write(buf[:n]); err != nil {
				return err
			}
			if progress != nil {
				progress(int64(n))
			}
		}
		if readErr == io.EOF {
			return nil
		}
		if readErr != nil {
			return readErr
		}
	}
}
