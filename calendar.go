package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

type CalendarWidget struct {
	app    fyne.App
	window fyne.Window
	notify func(title, content string)

	month      time.Time
	titleLabel *widget.Label
	grid       *fyne.Container
	status     *widget.Label
	summary    *widget.Label
}

type CalendarDay struct {
	Date      time.Time
	Day       int
	IsCurrent bool
}

func newCalendarWidget(a fyne.App, notify func(title, content string)) *CalendarWidget {
	w := a.NewWindow("Worklog Calendar")
	w.SetIcon(appIconResource())
	w.Resize(fyne.NewSize(calendarWindowWidth, calendarWindowHeight))
	w.SetFixedSize(true)
	w.CenterOnScreen()

	title := widget.NewLabel("")

	c := &CalendarWidget{
		app:        a,
		window:     w,
		notify:     notify,
		month:      currentMonthStart(),
		titleLabel: title,
		grid:       container.NewGridWithColumns(7),
		status:     widget.NewLabel("This month only."),
		summary:    widget.NewLabel(""),
	}

	refreshBtn := widget.NewButton("Refresh", func() {
		if err := c.refresh(); err != nil {
			c.notify("Tiny Worklog", "Could not refresh calendar.")
		}
	})

	openLogBtn := widget.NewButton("Open log", func() {
		c.month = currentMonthStart()

		path, err := ensureMonthLog(c.month)
		if err != nil {
			c.notify("Tiny Worklog", "Could not open current month log.")
			return
		}

		if err := openPath(path); err != nil {
			c.notify("Tiny Worklog", "Could not open current month log.")
			return
		}
	})

	w.SetContent(container.NewBorder(
		container.NewVBox(title, c.summary),
		container.NewVBox(
			c.status,
			container.NewGridWithColumns(2, refreshBtn, openLogBtn),
		),
		nil,
		nil,
		c.grid,
	))

	w.SetCloseIntercept(func() {
		w.Hide()
	})

	if err := c.refresh(); err != nil {
		c.notify("Tiny Worklog", "Could not load calendar.")
	}

	return c
}

func (c *CalendarWidget) Show() {
	if err := c.refresh(); err != nil {
		c.notify("Tiny Worklog", "Could not refresh calendar.")
	}

	c.window.CenterOnScreen()
	c.window.Show()
	c.window.RequestFocus()
}

func (c *CalendarWidget) Hide() {
	c.window.Hide()
}

func (c *CalendarWidget) refresh() error {
	// Current-month only.
	// Important: if app stays open from May 31 -> Jun 1,
	// this makes calendar move to June automatically.
	c.month = currentMonthStart()
	c.titleLabel.SetText(fmt.Sprintf("Worklog Calendar - %s", c.month.Format("January 2006")))

	loggedDays, err := loadMonthWorklogDays(c.month)
	if err != nil {
		return err
	}

	c.grid.Objects = nil

	weekdays := []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}
	for _, day := range weekdays {
		c.grid.Add(widget.NewLabel(day))
	}

	today := truncateDate(time.Now())
	cells := generateMonthCalendar(c.month)

	loggedCount := 0
	missingCount := 0

	for _, cell := range cells {
		cell := cell

		if !cell.IsCurrent {
			c.grid.Add(widget.NewLabel(""))
			continue
		}

		date := truncateDate(cell.Date)
		key := date.Format("2006-01-02")

		hasLog := loggedDays[key]
		isWeekend := date.Weekday() == time.Saturday || date.Weekday() == time.Sunday
		isFuture := date.After(today)
		isToday := date.Equal(today)

		statusIcon := "·"

		switch {
		case hasLog:
			statusIcon = "✅"
			loggedCount++

		case isFuture:
			statusIcon = "·"

		case isToday && !isWeekend:
			// Today is not counted as missing yet.
			statusIcon = "⏳"

		case isWeekend:
			statusIcon = "💤"

		default:
			statusIcon = "❌"
			missingCount++
		}

		if isToday {
			statusIcon = "📍" + statusIcon
		}

		clickedDate := date
		clickedHasLog := hasLog
		clickedIsWeekend := isWeekend
		clickedIsFuture := isFuture
		clickedIsToday := isToday

		btn := widget.NewButton(fmt.Sprintf("%d\n%s", cell.Day, statusIcon), func() {
			c.setDayStatus(clickedDate, clickedHasLog, clickedIsWeekend, clickedIsFuture, clickedIsToday)
			c.showEditDialog(clickedDate)
		})

		c.grid.Add(btn)
	}

	c.summary.SetText(fmt.Sprintf("Logged: %d | Missing weekday: %d", loggedCount, missingCount))
	c.status.SetText("✅ logged | ❌ missing | ⏳ today | 💤 weekend | · upcoming | 📍 today mark")

	c.grid.Refresh()
	return nil
}

func (c *CalendarWidget) setDayStatus(date time.Time, hasLog bool, isWeekend bool, isFuture bool, isToday bool) {
	formatted := date.Format("Mon, 02 Jan 2006")

	switch {
	case hasLog:
		c.status.SetText(formatted + " has worklog.")

	case isFuture:
		c.status.SetText(formatted + " is upcoming.")

	case isToday:
		c.status.SetText(formatted + " is today. No worklog yet.")

	case isWeekend:
		c.status.SetText(formatted + " is weekend.")

	default:
		c.status.SetText(formatted + " has no worklog yet.")
	}
}

func (c *CalendarWidget) showEditDialog(date time.Time) {
	rawLog, err := getRawDayLog(date)
	if err != nil {
		c.notify("Tiny Worklog", "Could not load log for this date.")
		return
	}

	editInput := widget.NewMultiLineEntry()
	editInput.SetText(rawLog)
	editInput.SetPlaceHolder("Enter worklog for " + date.Format("02 Jan 2006") + "...\nLeave empty to delete.")
	editInput.Wrapping = fyne.TextWrapWord
	editInput.SetMinRowsVisible(8)

	inputBox := container.NewGridWrap(
		fyne.NewSize(380, 200),
		editInput,
	)

	title := widget.NewLabel(fmt.Sprintf("Edit Log: %s", date.Format("Mon, 02 Jan 2006")))
	content := container.NewVBox(title, inputBox)

	d := dialog.NewCustomConfirm(
		"Edit Worklog",
		"Save Changes",
		"Cancel",
		content,
		func(confirm bool) {
			if !confirm {
				return
			}

			if err := replaceDayLog(date, editInput.Text); err != nil {
				c.notify("Tiny Worklog", "Failed to save log.")
			} else {
				c.notify("Tiny Worklog", "Log updated successfully.")
				c.refresh()
			}
		},
		c.window,
	)

	d.Resize(fyne.NewSize(420, 300))
	d.Show()
	c.window.Canvas().Focus(editInput)
}

func currentMonthStart() time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)
}

func generateMonthCalendar(month time.Time) []CalendarDay {
	firstDay := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, time.Local)
	totalDays := daysInMonth(month.Year(), month.Month())
	startOffset := mondayBasedWeekday(firstDay)

	cells := make([]CalendarDay, 0, 42)

	for i := 0; i < startOffset; i++ {
		cells = append(cells, CalendarDay{
			IsCurrent: false,
		})
	}

	for day := 1; day <= totalDays; day++ {
		date := time.Date(month.Year(), month.Month(), day, 0, 0, 0, 0, time.Local)

		cells = append(cells, CalendarDay{
			Date:      date,
			Day:       day,
			IsCurrent: true,
		})
	}

	for len(cells) < 42 {
		cells = append(cells, CalendarDay{
			IsCurrent: false,
		})
	}

	return cells
}

func daysInMonth(year int, month time.Month) int {
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.Local).Day()
}

func mondayBasedWeekday(t time.Time) int {
	// Go default: Sunday = 0.
	// This converts to: Monday = 0, Tuesday = 1, ..., Sunday = 6.
	return (int(t.Weekday()) + 6) % 7
}

func truncateDate(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local)
}

func loadMonthWorklogDays(month time.Time) (map[string]bool, error) {
	days := map[string]bool{}

	logDir, err := getLogDir()
	if err != nil {
		return days, err
	}

	fileName := month.Format("200601") + "_daily.txt"
	filePath := filepath.Join(logDir, fileName)

	contentBytes, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return days, nil
		}

		return days, err
	}

	content := strings.ReplaceAll(string(contentBytes), "\r\n", "\n")
	lines := strings.Split(content, "\n")

	dateRegex := regexp.MustCompile(`^\d{2}/\d{2}/\d{4}\s*$`)

	currentKey := ""
	currentHasBullet := false

	flush := func() {
		if currentKey != "" && currentHasBullet {
			days[currentKey] = true
		}
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if dateRegex.MatchString(trimmed) {
			flush()

			currentKey = ""
			currentHasBullet = false

			parsedDate, err := time.ParseInLocation("02/01/2006", trimmed, time.Local)
			if err != nil {
				continue
			}

			if parsedDate.Year() == month.Year() && parsedDate.Month() == month.Month() {
				currentKey = parsedDate.Format("2006-01-02")
			}

			continue
		}

		if currentKey == "" {
			continue
		}

		if strings.HasPrefix(trimmed, "*") && strings.TrimSpace(strings.TrimPrefix(trimmed, "*")) != "" {
			currentHasBullet = true
		}
	}

	flush()

	return days, nil
}
