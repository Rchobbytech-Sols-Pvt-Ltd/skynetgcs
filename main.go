package main

import (
	"context"
	"embed"
	"log"

	"github.com/jhakrishan20/skynetgcs/internal/activation"
	"github.com/jhakrishan20/skynetgcs/internal/launcher"
	"github.com/jhakrishan20/skynetgcs/internal/storage"
	"github.com/jhakrishan20/skynetgcs/internal/updater"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

type App struct {
	ctx     context.Context
	manager *launcher.Manager
}

func NewApp() *App {
	return &App{manager: launcher.NewManager()}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) shutdown(ctx context.Context) {
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

func (a *App) CheckForUpdates() (*updater.ReleaseInfo, error) {
	return updater.CheckLatest()
}

func (a *App) DownloadUpdate(release *updater.ReleaseInfo) error {
	return updater.DownloadAndApply(release)
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
	app := NewApp()

	err := wails.Run(&options.App{
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
