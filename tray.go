package main

import (
	"os"
	"time"

	"fyne.io/fyne/v2"
	desktop "fyne.io/fyne/v2/driver/desktop"
)

func setupSystemTray(
	a fyne.App,
	w fyne.Window,
	showPopup func(reason string),
	notify func(title, content string),
) {
	desk, ok := a.(desktop.App)
	if !ok {
		return
	}

	quitAfterNotify := func(title, content string) {
		notify(title, content)

		time.AfterFunc(1200*time.Millisecond, func() {
			fyne.Do(func() {
				a.Quit()
			})
		})
	}

	var refreshMenu func()

	refreshMenu = func() {
		autoStartEnabled := isAutoStartEnabled()

		enableLabel := "[ ] Enable auto-start"
		disableLabel := "[ ] Disable auto-start"

		if autoStartEnabled {
			enableLabel = "[x] Enable auto-start"
		} else {
			disableLabel = "[x] Disable auto-start"
		}

		menu := fyne.NewMenu("Tiny Worklog",
			fyne.NewMenuItem("Quick drop", func() {
				showPopup("Manual drop")
			}),
			fyne.NewMenuItemSeparator(),
			fyne.NewMenuItem("Open current month log", func() {
				path, err := ensureCurrentMonthLog(time.Now())
				if err != nil {
					notify("Tiny Worklog", "Could not open current month log.")
					return
				}

				if err := openPath(path); err != nil {
					notify("Tiny Worklog", "Could not open current month log.")
					return
				}
			}),
			fyne.NewMenuItem("Open logs folder", func() {
				logDir, err := getLogDir()
				if err != nil {
					notify("Tiny Worklog", "Could not find logs folder.")
					return
				}

				if err := os.MkdirAll(logDir, 0755); err != nil {
					notify("Tiny Worklog", "Could not open logs folder.")
					return
				}

				if err := openPath(logDir); err != nil {
					notify("Tiny Worklog", "Could not open logs folder.")
					return
				}
			}),
			fyne.NewMenuItemSeparator(),
			fyne.NewMenuItem(enableLabel, func() {
				if err := setAutoStart(true); err != nil {
					notify("Tiny Worklog", "Could not enable auto-start.")
					return
				}

				notify("Tiny Worklog", "Auto-start enabled.")
				refreshMenu()
			}),
			fyne.NewMenuItem(disableLabel, func() {
				if err := setAutoStart(false); err != nil {
					notify("Tiny Worklog", "Could not disable auto-start.")
					return
				}

				notify("Tiny Worklog", "Auto-start disabled.")
				refreshMenu()
			}),
			fyne.NewMenuItemSeparator(),
			fyne.NewMenuItem("Quit and disable auto-start", func() {
				if err := setAutoStart(false); err != nil {
					notify("Tiny Worklog", "Could not disable auto-start.")
					return
				}

				quitAfterNotify("Tiny Worklog", "Auto-start disabled. Quitting app.")
			}),
		)

		desk.SetSystemTrayMenu(menu)
		desk.SetSystemTrayWindow(w)
	}

	refreshMenu()
}
