package main

import (
	"fmt"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	desktop "fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

type tinyWorklogApp struct {
	fyneApp fyne.App
	window  fyne.Window
	input   *widget.Entry
	status  *widget.Label
}

func newTinyWorklogApp() *tinyWorklogApp {
	a := app.NewWithID(appID)
	a.SetIcon(appIconResource())

	w := a.NewWindow(appTitle)
	w.SetIcon(appIconResource())
	w.Resize(fyne.NewSize(windowWidth, windowHeight))
	w.SetFixedSize(true)
	w.CenterOnScreen()

	input := widget.NewMultiLineEntry()
	input.SetPlaceHolder("Brain dump here. One update per line.\n\nExample:\nDaily standup with...\nChecked QA feedback\nSupported ops on...")
	input.Wrapping = fyne.TextWrapWord
	input.SetMinRowsVisible(8)

	status := widget.NewLabel("Ready when you are.")

	worklog := &tinyWorklogApp{
		fyneApp: a,
		window:  w,
		input:   input,
		status:  status,
	}

	todoWidget := newTodoWidget(a, worklog.notify)
	calendarWidget := newCalendarWidget(a, worklog.notify)
	consoleWidget := initConsoleWidget(a)
	LogToConsole("Tiny Worklog App initialized.")

	title := widget.NewLabel("Drop your work update real quick.")

	saveBtn := widget.NewButton("Log it", worklog.saveInput)
	saveBtn.Importance = widget.HighImportance

	skipBtn := widget.NewButton("Not now", func() {
		LogToConsole("User skipped the prompt.")
		worklog.notify("Skipped for now", "No worries. Catch the next check-in.")
		worklog.hidePopup()
	})

	buttonStack := container.NewGridWithColumns(
		1,
		saveBtn,
		skipBtn,
	)

	worklog.setupShortcuts()

	w.SetContent(container.NewBorder(
		container.NewVBox(title),
		container.NewVBox(status, buttonStack),
		nil,
		nil,
		input,
	))

	w.SetCloseIntercept(func() {
		worklog.hidePopup()
	})

	setupSystemTray(
		a,
		w,
		worklog.showPopup,
		worklog.notify,
		todoWidget.Show,
		todoWidget.Hide,
		calendarWidget.Show,
		calendarWidget.Hide,
		consoleWidget.Show,
		consoleWidget.Hide,
	)

	startScheduler(reminderTimes, func(targetTime string) {
		fyne.Do(func() {
			worklog.showPopup("It's " + targetTime + " - quick work dump?")
		})
	})

	go PullFromGitHub(time.Now())

	return worklog
}

func (t *tinyWorklogApp) Run() {
	t.window.Hide()
	t.fyneApp.Run()
}

func (t *tinyWorklogApp) showPopup(reason string) {
	playPopupSFX()

	t.input.SetText("")
	t.status.SetText(reason)

	t.window.CenterOnScreen()
	t.window.Show()
	t.window.RequestFocus()
	t.window.Canvas().Focus(t.input)
}

func (t *tinyWorklogApp) hidePopup() {
	t.input.SetText("")
	t.window.Hide()
}

func (t *tinyWorklogApp) notify(title, content string) {
	t.fyneApp.SendNotification(&fyne.Notification{
		Title:   title,
		Content: content,
	})
}

func (t *tinyWorklogApp) saveInput() {
	text := strings.TrimSpace(t.input.Text)
	if text == "" {
		t.status.SetText("Nothing to save yet.")
		t.notify("Tiny Worklog", "No update entered. Nothing logged.")
		return
	}

	entries := normalizeEntries(text)
	if len(entries) == 0 {
		t.status.SetText("Nothing to save yet.")
		t.notify("Tiny Worklog", "No update entered. Nothing logged.")
		return
	}

	if err := saveWorkLog(text, time.Now()); err != nil {
		LogToConsole("Failed to save worklog locally: %v", err)
		dialog.ShowError(err, t.window)
		t.notify("Tiny Worklog", "Couldn't save that. Try again?")
		return
	}

	LogToConsole("Saved %d new entries locally.", len(entries))

	t.status.SetText("Logged at " + time.Now().Format("15:04"))
	t.notify("Logged. Nice.", fmt.Sprintf("%d update(s) saved.", len(entries)))

	t.hidePopup()
}

func (t *tinyWorklogApp) setupShortcuts() {
	addSaveShortcut := func(key fyne.KeyName) {
		t.window.Canvas().AddShortcut(&desktop.CustomShortcut{
			KeyName:  key,
			Modifier: fyne.KeyModifierControl,
		}, func(shortcut fyne.Shortcut) {
			t.saveInput()
		})
	}

	addSaveShortcut(fyne.KeyEnter)
	addSaveShortcut(fyne.KeyReturn)

	t.window.Canvas().SetOnTypedKey(func(e *fyne.KeyEvent) {
		if e.Name == fyne.KeyEscape {
			t.hidePopup()
		}
	})
}
