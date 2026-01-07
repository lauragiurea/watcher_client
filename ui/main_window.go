package ui

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"watcher-client/api"
)

type MainWindow struct {
	App    fyne.App
	Window fyne.Window
	Client *api.Client

	monitors      []api.Monitor
	list          *widget.List
	selectedIndex int
}

func NewMainWindow(a fyne.App, client *api.Client) *MainWindow {
	w := a.NewWindow("Watcher – Desktop Client")

	mw := &MainWindow{
		App:           a,
		Window:        w,
		Client:        client,
		selectedIndex: -1,
	}

	var historyBtn *widget.Button
	var detailsBtn *widget.Button
	var deleteBtn *widget.Button

	updateSelectionButtons := func() {}

	mw.list = widget.NewList(
		func() int { return len(mw.monitors) },
		func() fyne.CanvasObject {
			return widget.NewLabel("monitor")
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			m := mw.monitors[i]
			label := o.(*widget.Label)
			text := fmt.Sprintf("%s (%s)", m.Name, m.URL)
			if !m.Active {
				text += " [inactive]"
			}
			label.SetText(text)
		},
	)

	mw.list.OnSelected = func(id widget.ListItemID) {
		mw.selectedIndex = int(id)
		updateSelectionButtons()
	}

	mw.list.OnUnselected = func(id widget.ListItemID) {
		if mw.selectedIndex == int(id) {
			mw.selectedIndex = -1
		}
		updateSelectionButtons()
	}
	addBtn := widget.NewButton("Add monitor", func() {
		mw.showAddMonitorDialog()
	})
	deleteBtn = widget.NewButton("Delete", func() {
		if mw.selectedIndex < 0 || mw.selectedIndex >= len(mw.monitors) {
			mw.showInfo("No monitor selected")
			return
		}
		m := mw.monitors[mw.selectedIndex]
		mw.confirmDelete(m)
	})

	historyBtn = widget.NewButton("History", func() {
		if mw.selectedIndex < 0 || mw.selectedIndex >= len(mw.monitors) {
			mw.showInfo("No monitor selected")
			return
		}
		m := mw.monitors[mw.selectedIndex]
		ShowHistoryWindow(mw.App, mw.Client, m)
	})
	detailsBtn = widget.NewButton("Edit", func() {
		if mw.selectedIndex < 0 || mw.selectedIndex >= len(mw.monitors) {
			mw.showInfo("No monitor selected")
			return
		}
		m := mw.monitors[mw.selectedIndex]
		mw.showMonitorDetails(m, mw.selectedIndex)
	})

	historyBtn.Disable()
	detailsBtn.Disable()
	deleteBtn.Disable()

	updateSelectionButtons = func() {
		hasSelection := mw.selectedIndex >= 0 && mw.selectedIndex < len(mw.monitors)
		if hasSelection {
			historyBtn.Enable()
			detailsBtn.Enable()
			deleteBtn.Enable()
		} else {
			historyBtn.Disable()
			detailsBtn.Disable()
			deleteBtn.Disable()
		}
	}

	topBar := container.NewHBox(addBtn, deleteBtn, historyBtn, detailsBtn)
	content := container.NewBorder(topBar, nil, nil, nil, mw.list)

	w.SetContent(content)
	w.Resize(fyne.NewSize(900, 600))

	mw.loadMonitors()
	updateSelectionButtons()
	return mw
}

func (mw *MainWindow) loadMonitors() {
	ms, err := mw.Client.ListMonitors()
	if err != nil {
		mw.showError("Failed to load monitors: " + err.Error())
		return
	}
	mw.monitors = ms
	mw.selectedIndex = -1
	mw.list.UnselectAll()
	mw.list.Refresh()
}

func (mw *MainWindow) showError(msg string) {
	dialog.ShowError(errors.New(msg), mw.Window)
}

func (mw *MainWindow) showInfo(msg string) {
	dialog.ShowInformation("Info", msg, mw.Window)
}

func (mw *MainWindow) showAddMonitorDialog() {
	nameEntry := widget.NewEntry()
	urlEntry := widget.NewEntry()
	cssEntry := widget.NewEntry()
	freqEntry := widget.NewEntry()
	freqEntry.SetText("300") // default 5 minutes

	renderCheck := widget.NewCheck("Render JS (headless browser)", nil)
	emailCheck := widget.NewCheck("Notify by email", nil)
	emailCheck.SetChecked(true)

	emailAddrEntry := widget.NewEntry()
	emailAddrEntry.SetPlaceHolder("your@email.com")

	form := dialog.NewForm(
		"Add monitor",
		"Create",
		"Cancel",
		[]*widget.FormItem{
			widget.NewFormItem("Name", nameEntry),
			widget.NewFormItem("URL", urlEntry),
			widget.NewFormItem("CSS selector", cssEntry),
			widget.NewFormItem("Frequency (seconds)", freqEntry),
			widget.NewFormItem("", renderCheck),
			widget.NewFormItem("", emailCheck),
			widget.NewFormItem("Notification email", emailAddrEntry),
		},
		func(confirmed bool) {
			if !confirmed {
				return
			}
			url := urlEntry.Text
			if url == "" {
				mw.showError("URL is required")
				return
			}
			name := nameEntry.Text
			if name == "" {
				name = url
			}

			freq, err := strconv.Atoi(freqEntry.Text)
			if err != nil || freq <= 0 {
				freq = 300
			}

			var css *string
			if cssEntry.Text != "" {
				c := cssEntry.Text
				css = &c
			}

			notifyEmail := emailCheck.Checked
			emailAddr := strings.TrimSpace(emailAddrEntry.Text)
			if notifyEmail && emailAddr == "" {
				mw.showError("Please enter an email address for notifications")
				return
			}

			req := api.CreateMonitorReq{
				Name:             name,
				URL:              url,
				CSSSelector:      css,
				RenderJS:         renderCheck.Checked,
				FrequencySeconds: freq,
				NotifyEmail:      notifyEmail,
				NotifyEmailAddr:  emailAddr,
			}

			m, err := mw.Client.CreateMonitor(req)
			if err != nil {
				mw.showError("Create failed: " + err.Error())
				return
			}
			mw.monitors = append([]api.Monitor{*m}, mw.monitors...)
			mw.selectedIndex = 0
			mw.list.Refresh()
			mw.list.Select(0)
		},
		mw.Window,
	)
	form.Resize(fyne.NewSize(450, 380))
	form.Show()
}

func (mw *MainWindow) confirmDelete(m api.Monitor) {
	dialog.ShowConfirm(
		"Delete monitor",
		"Delete monitor '"+m.Name+"'?",
		func(ok bool) {
			if !ok {
				return
			}
			if err := mw.Client.DeleteMonitor(m.ID); err != nil {
				mw.showError("Delete failed: " + err.Error())
				return
			}
			mw.loadMonitors()
		},
		mw.Window,
	)
}

func (mw *MainWindow) showMonitorDetails(m api.Monitor, index int) {
	freqEntry := widget.NewEntry()
	freqEntry.SetText(strconv.Itoa(m.FrequencySeconds))

	activeCheck := widget.NewCheck("Monitor is active", nil)
	activeCheck.SetChecked(m.Active)

	form := dialog.NewForm(
		"Monitor details – "+m.Name,
		"Save",
		"Cancel",
		[]*widget.FormItem{
			widget.NewFormItem("Frequency (seconds)", freqEntry),
			widget.NewFormItem("", activeCheck),
		},
		func(confirmed bool) {
			if !confirmed {
				return
			}
			freq, err := strconv.Atoi(strings.TrimSpace(freqEntry.Text))
			if err != nil || freq <= 0 {
				mw.showError("Please enter a valid positive frequency (seconds)")
				return
			}

			req := api.UpdateMonitorReq{
				FrequencySeconds: freq,
				Active:           activeCheck.Checked,
			}

			updated, err := mw.Client.UpdateMonitor(m.ID, req)
			if err != nil {
				mw.showError("Update failed: " + err.Error())
				return
			}

			mw.monitors[index] = *updated
			mw.list.Refresh()
		},
		mw.Window,
	)
	form.Resize(fyne.NewSize(350, 220))
	form.Show()
}
