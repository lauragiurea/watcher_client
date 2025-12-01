package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"watcher-client/api"
)

func ShowHistoryWindow(a fyne.App, client *api.Client, m api.Monitor) {
	w := a.NewWindow("History – " + m.Name)

	var changes []api.ChangeEvent
	list := widget.NewList(
		func() int { return len(changes) },
		func() fyne.CanvasObject {
			return widget.NewLabel("change")
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			c := changes[i]
			lbl := o.(*widget.Label)
			lbl.SetText(fmt.Sprintf("%s", c.CreatedAt.Format("2006-01-02 15:04:05")))
		},
	)

	list.OnSelected = func(id widget.ListItemID) {
		if id < 0 || id >= widget.ListItemID(len(changes)) {
			return
		}
		ShowChangeDetailWindow(a, changes[id], m)
	}

	refresh := func() {
		evts, err := client.ListChanges(m.ID)
		if err != nil {
			dialog.ShowError(err, w)
			return
		}
		changes = evts
		list.Refresh()
	}

	refreshBtn := widget.NewButton("Refresh", func() { refresh() })

	content := container.NewBorder(refreshBtn, nil, nil, nil, list)

	w.SetContent(content)
	w.Resize(fyne.NewSize(600, 400))
	w.Show()

	refresh()
}
