package main

import (
	"embed"
	"fmt"
	"log"
	"os"

	"github.com/user/vpm/internal/daemon"
	"github.com/user/vpm/internal/tray"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// CLI modes (headless / boot-persistence) run before the GUI is created.
	switch {
	case hasArg("--daemon"):
		if err := daemon.Run(); err != nil {
			log.Fatalln("daemon:", err)
		}
		return
	case hasArg("--install-daemon"):
		if err := tray.SetDaemonAutostart(true); err != nil {
			log.Fatalln("install-daemon:", err)
		}
		fmt.Println("VPM daemon registered to start at login (vpm --daemon).")
		return
	case hasArg("--uninstall-daemon"):
		if err := tray.SetDaemonAutostart(false); err != nil {
			log.Fatalln("uninstall-daemon:", err)
		}
		fmt.Println("VPM daemon autostart removed.")
		return
	}

	app := NewApp()
	err := wails.Run(&options.App{
		Title:            "Vibe Port Manager",
		Width:            1100,
		Height:           720,
		MinWidth:         360,
		MinHeight:        44,
		Frameless:        true, // custom dark titlebar (components/titlebar.js)
		AssetServer:      &assetserver.Options{Assets: assets},
		BackgroundColour: &options.RGBA{R: 3, G: 7, B: 18, A: 1}, // gray-950, matches the UI
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		OnBeforeClose:    app.beforeClose,
		Bind:             []interface{}{app},
	})
	if err != nil {
		log.Println("Error:", err.Error())
		os.Exit(1)
	}
}

// hasArg reports whether flag appears in the process arguments.
func hasArg(flag string) bool {
	for _, a := range os.Args[1:] {
		if a == flag {
			return true
		}
	}
	return false
}
