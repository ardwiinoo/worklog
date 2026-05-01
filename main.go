package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	desktop "fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

const (
	appID        = "com.local.worklog"
	appTitle     = "Tiny Worklog"
	logDirName   = "logs"
	windowWidth  = 480
	windowHeight = 300
)

var reminderTimes = []string{
	"10:30",
	"13:30",
	"16:30",
}

func main() {
	a := app.NewWithID(appID)

	w := a.NewWindow(appTitle)
	w.Resize(fyne.NewSize(windowWidth, windowHeight))
	w.SetFixedSize(true)
	w.CenterOnScreen()

	input := widget.NewMultiLineEntry()
	input.SetPlaceHolder("Brain dump here. One update per line.\n\nExample:\nDaily standup with...\nChecked QA feedback\nSupported ops on...")
	input.Wrapping = fyne.TextWrapWord
	input.SetMinRowsVisible(8)

	status := widget.NewLabel("Ready when you are.")
	title := widget.NewLabel("Drop your work update real quick.")

	showPopup := func(reason string) {
		input.SetText("")
		status.SetText(reason)

		w.CenterOnScreen()
		w.Show()
		w.RequestFocus()
		w.Canvas().Focus(input)
	}

	hidePopup := func() {
		input.SetText("")
		w.Hide()
	}

	notify := func(title, content string) {
		a.SendNotification(&fyne.Notification{
			Title:   title,
			Content: content,
		})
	}

	saveInput := func() {
		text := strings.TrimSpace(input.Text)
		if text == "" {
			status.SetText("Nothing to save yet.")
			notify("Tiny Worklog", "No update entered. Nothing logged.")
			return
		}

		entries := normalizeEntries(text)
		if len(entries) == 0 {
			status.SetText("Nothing to save yet.")
			notify("Tiny Worklog", "No update entered. Nothing logged.")
			return
		}

		if err := saveWorkLog(text, time.Now()); err != nil {
			dialog.ShowError(err, w)
			notify("Tiny Worklog", "Couldn't save that. Try again?")
			return
		}

		status.SetText("Logged at " + time.Now().Format("15:04"))
		notify("Logged. Nice.", fmt.Sprintf("%d update(s) saved.", len(entries)))

		hidePopup()
	}

	saveBtn := widget.NewButton("Log it", saveInput)
	saveBtn.Importance = widget.HighImportance

	skipBtn := widget.NewButton("Not now", func() {
		notify("Skipped for now", "No worries. Catch the next check-in.")
		hidePopup()
	})

	buttonStack := container.NewGridWithColumns(
		1,
		saveBtn,
		skipBtn,
	)

	w.SetContent(container.NewBorder(
		container.NewVBox(title),
		container.NewVBox(buttonStack),
		nil,
		nil,
		container.NewVBox(input, status),
	))

	// Klik X jangan matiin app, cukup hide.
	w.SetCloseIntercept(func() {
		w.Hide()
	})

	setupSystemTray(a, w, showPopup)

	startScheduler(reminderTimes, func(targetTime string) {
		fyne.Do(func() {
			showPopup("It's " + targetTime + " - quick work dump?")
		})
	})

	// Start hidden. App tetap hidup sampai user pilih Exit dari tray.
	w.Hide()
	a.Run()
}

func setupSystemTray(a fyne.App, w fyne.Window, showPopup func(reason string)) {
	desk, ok := a.(desktop.App)
	if !ok {
		return
	}

	menu := fyne.NewMenu("Tiny Worklog",
		fyne.NewMenuItem("Show", func() {
			showPopup("Manual drop")
		}),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Quit", func() {
			a.Quit()
		}),
	)

	desk.SetSystemTrayMenu(menu)

	// Fyne 2.7+: klik tray icon bisa bantu show window di OS yang support.
	desk.SetSystemTrayWindow(w)
}

func startScheduler(times []string, onReminder func(targetTime string)) {
	go func() {
		triggered := map[string]bool{}

		check := func(now time.Time) {
			currentHHMM := now.Format("15:04")
			today := now.Format("2006-01-02")

			for _, target := range times {
				if currentHHMM != target {
					continue
				}

				key := today + " " + target
				if triggered[key] {
					continue
				}

				triggered[key] = true
				onReminder(target)
			}

			// Cleanup trigger hari sebelumnya biar map gak numpuk.
			for key := range triggered {
				if !strings.HasPrefix(key, today) {
					delete(triggered, key)
				}
			}
		}

		// Cek langsung saat app baru hidup.
		check(time.Now())

		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for now := range ticker.C {
			check(now)
		}
	}()
}

func saveWorkLog(rawInput string, now time.Time) error {
	entries := normalizeEntries(rawInput)
	if len(entries) == 0 {
		return nil
	}

	if err := os.MkdirAll(logDirName, 0755); err != nil {
		return err
	}

	fileName := now.Format("200601") + "_daily.txt"
	filePath := filepath.Join(logDirName, fileName)

	dateHeader := now.Format("02/01/2006")

	bullets := make([]string, 0, len(entries))
	for _, entry := range entries {
		bullets = append(bullets, "* "+entry)
	}

	contentBytes, err := os.ReadFile(filePath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	content := string(contentBytes)

	if strings.TrimSpace(content) == "" {
		newContent := dateHeader + "\n" + strings.Join(bullets, "\n") + "\n"
		return os.WriteFile(filePath, []byte(newContent), 0644)
	}

	newContent := insertBullets(content, dateHeader, bullets)
	return os.WriteFile(filePath, []byte(newContent), 0644)
}

func normalizeEntries(rawInput string) []string {
	normalized := strings.ReplaceAll(rawInput, "\r\n", "\n")
	lines := strings.Split(normalized, "\n")

	entries := make([]string, 0, len(lines))

	for _, line := range lines {
		entry := strings.TrimSpace(line)
		if entry == "" {
			continue
		}

		// Kalau user ngetik "* sesuatu", jangan jadi "** sesuatu".
		entry = strings.TrimPrefix(entry, "*")
		entry = strings.TrimSpace(entry)

		if entry != "" {
			entries = append(entries, entry)
		}
	}

	return entries
}

func insertBullets(content, dateHeader string, bullets []string) string {
	normalized := strings.ReplaceAll(content, "\r\n", "\n")
	lines := strings.Split(normalized, "\n")

	dateRegex := regexp.MustCompile(`^\d{2}/\d{2}/\d{4}\s*$`)

	dateIndex := -1
	for i, line := range lines {
		if strings.TrimSpace(line) == dateHeader {
			dateIndex = i
			break
		}
	}

	if dateIndex == -1 {
		trimmed := strings.TrimRight(normalized, "\n")
		return trimmed + "\n\n" + dateHeader + "\n" + strings.Join(bullets, "\n") + "\n"
	}

	insertAt := len(lines)

	for i := dateIndex + 1; i < len(lines); i++ {
		if dateRegex.MatchString(strings.TrimSpace(lines[i])) {
			insertAt = i
			break
		}
	}

	for insertAt > dateIndex+1 && strings.TrimSpace(lines[insertAt-1]) == "" {
		insertAt--
	}

	newLines := make([]string, 0, len(lines)+len(bullets))
	newLines = append(newLines, lines[:insertAt]...)
	newLines = append(newLines, bullets...)
	newLines = append(newLines, lines[insertAt:]...)

	return strings.TrimRight(strings.Join(newLines, "\n"), "\n") + "\n"
}
