// Command martis-desktop opens Martis in a native window via the system webview.
package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
)

//go:embed all:frontend
var assets embed.FS

//go:embed icon.png
var icon []byte

var version = "dev"

func main() {
	app := NewApp(version)
	err := wails.Run(&options.App{
		Title:            "Martis",
		Width:            1280,
		Height:           800,
		MinWidth:         880,
		MinHeight:        560,
		BackgroundColour: &options.RGBA{R: 12, G: 12, B: 13, A: 255},
		AssetServer:      &assetserver.Options{Assets: assets},
		Bind:             []any{app},
		OnStartup:        app.startup,
		Linux:            &linux.Options{Icon: icon, ProgramName: "martis"},
		Mac: &mac.Options{
			TitleBar: mac.TitleBarHiddenInset(),
			About:    &mac.AboutInfo{Title: "Martis", Message: "Lightweight REST client\n" + version, Icon: icon},
		},
	})
	if err != nil {
		log.Fatal(err)
	}
}
