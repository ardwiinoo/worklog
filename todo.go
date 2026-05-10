package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type TodoItem struct {
	ID        int64  `json:"id"`
	Text      string `json:"text"`
	Done      bool   `json:"done"`
	CreatedAt string `json:"created_at"`
}

type TodoState struct {
	Items []TodoItem `json:"items"`
}

type TodoWidget struct {
	app    fyne.App
	window fyne.Window
	notify func(title, content string)

	items   []TodoItem
	listBox *fyne.Container
	input   *widget.Entry
	status  *widget.Label
}

func newTodoWidget(a fyne.App, notify func(title, content string)) *TodoWidget {
	w := a.NewWindow("Today Focus")
	w.SetIcon(appIconResource())
	w.Resize(fyne.NewSize(todoWindowWidth, todoWindowHeight))
	w.SetFixedSize(true)
	w.CenterOnScreen()

	t := &TodoWidget{
		app:     a,
		window:  w,
		notify:  notify,
		listBox: container.NewVBox(),
		input:   widget.NewEntry(),
		status:  widget.NewLabel("Temporary todo list."),
	}

	t.input.SetPlaceHolder("Add priority task...")

	if err := t.load(); err != nil {
		t.notify("Tiny Worklog", "Could not load todo list.")
	}

	title := widget.NewLabel("Today Focus")
	subtitle := widget.NewLabel("Temporary priorities. Keep it light.")

	addBtn := widget.NewButton("Add", func() {
		t.addItem(t.input.Text)
	})
	addBtn.Importance = widget.HighImportance

	clearDoneBtn := widget.NewButton("Clear done", func() {
		t.clearDone()
	})

	inputRow := container.NewBorder(
		nil,
		nil,
		nil,
		addBtn,
		t.input,
	)

	scroll := container.NewVScroll(t.listBox)
	scroll.SetMinSize(fyne.NewSize(todoWindowWidth-30, 250))

	w.SetContent(container.NewBorder(
		container.NewVBox(title, subtitle),
		container.NewVBox(t.status, inputRow, clearDoneBtn),
		nil,
		nil,
		scroll,
	))

	w.SetCloseIntercept(func() {
		w.Hide()
	})

	t.refresh()

	return t
}

func (t *TodoWidget) Show() {
	t.window.CenterOnScreen()
	t.window.Show()
	t.window.RequestFocus()
	t.window.Canvas().Focus(t.input)
}

func (t *TodoWidget) Hide() {
	t.window.Hide()
}

func (t *TodoWidget) addItem(text string) {
	text = strings.TrimSpace(text)
	if text == "" {
		t.status.SetText("Nothing to add.")
		return
	}

	item := TodoItem{
		ID:        time.Now().UnixNano(),
		Text:      text,
		Done:      false,
		CreatedAt: time.Now().Format(time.RFC3339),
	}

	t.items = append(t.items, item)
	t.input.SetText("")
	t.status.SetText("Added.")

	if err := t.save(); err != nil {
		t.notify("Tiny Worklog", "Could not save todo list.")
		return
	}

	t.refresh()
}

func (t *TodoWidget) toggleItem(id int64, done bool) {
	for i := range t.items {
		if t.items[i].ID == id {
			t.items[i].Done = done
			break
		}
	}

	if err := t.save(); err != nil {
		t.notify("Tiny Worklog", "Could not save todo list.")
		return
	}

	t.refresh()
}

func (t *TodoWidget) deleteItem(id int64) {
	next := make([]TodoItem, 0, len(t.items))

	for _, item := range t.items {
		if item.ID != id {
			next = append(next, item)
		}
	}

	t.items = next
	t.status.SetText("Deleted.")

	if err := t.save(); err != nil {
		t.notify("Tiny Worklog", "Could not save todo list.")
		return
	}

	t.refresh()
}

func (t *TodoWidget) clearDone() {
	next := make([]TodoItem, 0, len(t.items))
	removed := 0

	for _, item := range t.items {
		if item.Done {
			removed++
			continue
		}

		next = append(next, item)
	}

	t.items = next
	t.status.SetText("Cleared done items.")

	if removed == 0 {
		t.status.SetText("No done items to clear.")
		return
	}

	if err := t.save(); err != nil {
		t.notify("Tiny Worklog", "Could not save todo list.")
		return
	}

	t.refresh()
}

func (t *TodoWidget) refresh() {
	t.listBox.Objects = nil

	if len(t.items) == 0 {
		t.listBox.Add(widget.NewLabel("No priorities yet. Add one below."))
		t.listBox.Refresh()
		return
	}

	for _, item := range t.items {
		itemID := item.ID
		itemText := item.Text
		itemDone := item.Done

		check := widget.NewCheck("", nil)
		check.SetChecked(itemDone)

		check.OnChanged = func(done bool) {
			t.toggleItem(itemID, done)
		}

		checkBox := container.NewGridWrap(
			fyne.NewSize(32, 32),
			check,
		)

		textLabel := widget.NewLabel(itemText)
		textLabel.Wrapping = fyne.TextWrapWord

		editBtn := widget.NewButtonWithIcon("", theme.DocumentCreateIcon(), func() {
			t.showEditDialog(itemID, itemText)
		})
		editBtn.Importance = widget.LowImportance

		deleteBtn := widget.NewButton("×", func() {
			t.deleteItem(itemID)
		})
		deleteBtn.Importance = widget.LowImportance

		actionBox := container.NewHBox(
			editBtn,
			deleteBtn,
		)

		actionBoxWrap := container.NewGridWrap(
			fyne.NewSize(76, 32),
			actionBox,
		)

		row := container.NewBorder(
			nil,
			nil,
			checkBox,
			actionBoxWrap,
			textLabel,
		)

		t.listBox.Add(row)
		t.listBox.Add(widget.NewSeparator())
	}

	t.listBox.Refresh()
}

func (t *TodoWidget) load() error {
	path, err := getTodoPath()
	if err != nil {
		return err
	}

	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			t.items = []TodoItem{}
			return nil
		}

		return err
	}

	if len(strings.TrimSpace(string(content))) == 0 {
		t.items = []TodoItem{}
		return nil
	}

	var state TodoState
	if err := json.Unmarshal(content, &state); err != nil {
		return err
	}

	t.items = state.Items
	return nil
}

func (t *TodoWidget) save() error {
	path, err := getTodoPath()
	if err != nil {
		return err
	}

	state := TodoState{
		Items: t.items,
	}

	content, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, content, 0644)
}

func getTodoPath() (string, error) {
	baseDir, err := getAppBaseDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(baseDir, todoFileName), nil
}

func (t *TodoWidget) updateItem(id int64, text string) {
	text = strings.TrimSpace(text)
	if text == "" {
		t.status.SetText("Todo cannot be empty.")
		return
	}

	for i := range t.items {
		if t.items[i].ID == id {
			t.items[i].Text = text
			break
		}
	}

	t.status.SetText("Updated.")

	if err := t.save(); err != nil {
		t.notify("Tiny Worklog", "Could not save todo list.")
		return
	}

	t.refresh()
}

func (t *TodoWidget) showEditDialog(id int64, currentText string) {
	editInput := widget.NewMultiLineEntry()
	editInput.SetText(currentText)
	editInput.SetPlaceHolder("Update todo...")
	editInput.Wrapping = fyne.TextWrapWord
	editInput.SetMinRowsVisible(3)

	inputBox := container.NewGridWrap(
		fyne.NewSize(300, 100),
		editInput,
	)

	content := container.NewVBox(
		widget.NewLabel("Edit priority task"),
		inputBox,
	)

	d := dialog.NewCustomConfirm(
		"Edit Todo",
		"Save",
		"Cancel",
		content,
		func(confirm bool) {
			if !confirm {
				return
			}

			t.updateItem(id, editInput.Text)
		},
		t.window,
	)

	d.Resize(fyne.NewSize(360, 230))
	d.Show()
}
