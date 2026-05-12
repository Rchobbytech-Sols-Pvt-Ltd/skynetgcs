package main

import (
	"context"
	"embed"
	"log"
	"net"
	"os"
	"sync"

	"github.com/Rchobbytech-Sols-Pvt-Ltd/skynetgcs/internal/activation"
	"github.com/Rchobbytech-Sols-Pvt-Ltd/skynetgcs/internal/launcher"
	"github.com/Rchobbytech-Sols-Pvt-Ltd/skynetgcs/internal/storage"
	"github.com/Rchobbytech-Sols-Pvt-Ltd/skynetgcs/internal/updater"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

type App struct {
	ctx          context.Context
	manager      *launcher.Manager
	updateMu     sync.Mutex
	updateCancel context.CancelFunc
	updateID     int
	updateStatus UpdateStatus
}

type UpdateStatus struct {
	Downloading     bool  `json:"downloading"`
	Percent         int   `json:"percent"`
	DownloadedBytes int64 `json:"downloaded_bytes"`
	TotalBytes      int64 `json:"total_bytes"`
}

func NewApp() *App {
	return &App{manager: launcher.NewManager()}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) shutdown(ctx context.Context) {
	a.CancelUpdate()
	a.manager.StopAll()
}

func (a *App) Activate(key string) (bool, error) {
	ok, err := activation.Activate(key)
	if err != nil {
		return false, err
	}
	if ok {
		if err := storage.SaveActivation(key); err != nil {
			return false, err
		}
	}
	return ok, nil
}

func (a *App) IsActivated() bool {
	return storage.IsActivated()
}

func (a *App) CheckForUpdates() (*updater.UpdateCheckResult, error) {
	return updater.CheckForUpdate()
}

func (a *App) DownloadUpdate(release *updater.ReleaseInfo) error {
	a.updateMu.Lock()
	if a.updateCancel != nil {
		a.updateCancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.updateID++
	updateID := a.updateID
	a.updateCancel = cancel
	a.updateStatus = UpdateStatus{Downloading: true}
	a.updateMu.Unlock()

	err := updater.DownloadAndApplyContext(ctx, release, func(downloadedBytes, totalBytes int64) {
		a.updateMu.Lock()
		if a.updateID == updateID {
			percent := 0
			if totalBytes > 0 {
				percent = int(downloadedBytes * 100 / totalBytes)
				if percent > 100 {
					percent = 100
				}
			}
			a.updateStatus = UpdateStatus{
				Downloading:     true,
				Percent:         percent,
				DownloadedBytes: downloadedBytes,
				TotalBytes:      totalBytes,
			}
		}
		a.updateMu.Unlock()
	})

	a.updateMu.Lock()
	if a.updateID == updateID {
		a.updateCancel = nil
		if err == nil {
			a.updateStatus.Percent = 100
			a.updateStatus.DownloadedBytes = a.updateStatus.TotalBytes
		}
		a.updateStatus.Downloading = false
	}
	a.updateMu.Unlock()

	return err
}

func (a *App) CancelUpdate() {
	a.updateMu.Lock()
	cancel := a.updateCancel
	a.updateCancel = nil
	a.updateID++
	a.updateStatus.Downloading = false
	a.updateMu.Unlock()

	if cancel != nil {
		log.Printf("[updater] Cancelling active update download")
		cancel()
	}
}

func (a *App) UpdateStatus() UpdateStatus {
	a.updateMu.Lock()
	defer a.updateMu.Unlock()
	return a.updateStatus
}

func (a *App) LaunchApps() ([]launcher.ChildStatus, error) {
	return a.manager.StartAll()
}

func (a *App) StopApps() {
	a.manager.StopAll()
}

func (a *App) AppsStatus() []launcher.ChildStatus {
	return a.manager.Status()
}

func main() {
	// Ensure only one instance of the app runs at a time by attempting to listen on a local port.
	// The OS will automatically release this lock if the process crashes or exits.
	l, err := net.Listen("tcp", "127.0.0.1:43215")
	if err != nil {
		// Another instance is likely running; exit silently.
		os.Exit(0)
	}
	defer l.Close()

	app := NewApp()

	err = wails.Run(&options.App{
		Title:  "Skynet GCS",
		Width:  1200,
		Height: 800,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup:  app.startup,
		OnShutdown: app.shutdown,
		Bind: []interface{}{
			app,
		},
	})
	if err != nil {
		log.Fatal(err)
	}
}
