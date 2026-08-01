package main

import (
	"fmt"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type ConsoleWidget struct {
	window fyne.Window
	logs   []string
	list   *widget.List
	mutex  sync.Mutex
}

var globalConsole *ConsoleWidget

func initConsoleWidget(a fyne.App) *ConsoleWidget {
	w := a.NewWindow("Sync Console")
	w.SetIcon(appIconResource())
	w.Resize(fyne.NewSize(500, 400))
	w.SetCloseIntercept(func() {
		w.Hide()
	})

	c := &ConsoleWidget{
		window: w,
		logs:   make([]string, 0),
	}

	c.list = widget.NewList(
		func() int {
			c.mutex.Lock()
			defer c.mutex.Unlock()
			return len(c.logs)
		},
		func() fyne.CanvasObject {
			label := widget.NewLabel("Template")
			label.Wrapping = fyne.TextWrapWord
			return label
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			c.mutex.Lock()
			defer c.mutex.Unlock()
			if i >= 0 && i < len(c.logs) {
				// Reverse order so newest is at the top
				o.(*widget.Label).SetText(c.logs[len(c.logs)-1-i])
			}
		},
	)

	w.SetContent(container.NewBorder(
		widget.NewLabel("Background Sync Logs:"),
		nil, nil, nil,
		c.list,
	))

	globalConsole = c
	return c
}

func (c *ConsoleWidget) Show() {
	c.window.CenterOnScreen()
	c.window.Show()
	c.window.RequestFocus()
}

func (c *ConsoleWidget) Hide() {
	c.window.Hide()
}

func LogToConsole(msg string, args ...interface{}) {
	if globalConsole == nil {
		return
	}

	formatted := fmt.Sprintf(msg, args...)
	timestamp := time.Now().Format("15:04:05")
	finalMsg := fmt.Sprintf("[%s] %s", timestamp, formatted)

	globalConsole.mutex.Lock()
	globalConsole.logs = append(globalConsole.logs, finalMsg)
	// Keep max 100 logs
	if len(globalConsole.logs) > 100 {
		globalConsole.logs = globalConsole.logs[1:]
	}
	globalConsole.mutex.Unlock()

	// Refresh UI on main thread
	fyne.Do(func() {
		if globalConsole.list != nil {
			globalConsole.list.Refresh()
		}
	})
}
