//go:build windows

package main

import "golang.org/x/sys/windows"

const mbIconAsterisk = 0x00000040

var (
	user32DLL       = windows.NewLazySystemDLL("user32.dll")
	messageBeepProc = user32DLL.NewProc("MessageBeep")
)

func playPopupSFX() {
	go func() {
		messageBeepProc.Call(uintptr(mbIconAsterisk))
	}()
}
