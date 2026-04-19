package main

import (
	"log"
	"os"

	"jpg-to-webp/backend/app"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

func main() {
	application := app.New()

		err := wails.Run(&options.App{
			Title:  "JPG to WEBP",
			Width:  960,
			Height: 720,
			AssetServer: &assetserver.Options{
				Assets: os.DirFS("frontend/dist"),
			},
		OnStartup: application.Startup,
		Bind: []interface{}{
			application,
		},
	})
	if err != nil {
		log.Fatalf("run wails app: %v", err)
	}
}
