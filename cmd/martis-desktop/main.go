// Command martis-desktop opens Martis in a native window via the system webview.
package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
)

//go:embed all:frontend
var assets embed.FS

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
		Mac: &mac.Options{
			TitleBar:             mac.TitleBarHiddenInset(),
			WebviewIsTransparent: false,
		},
	})
	if err != nil {
		log.Fatal(err)
	}
}
