package main

import (
	_ "embed"

	"fyne.io/fyne/v2"
)

//go:embed assets/icon.png
var appIconBytes []byte

func appIconResource() fyne.Resource {
	return fyne.NewStaticResource("icon.png", appIconBytes)
}
