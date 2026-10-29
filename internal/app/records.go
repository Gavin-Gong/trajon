package app

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/oneo-me/onevra/source/theme"
)

// Quick entry retains its "next blank after a saved review" behavior. Opening
// a specific historical record or popping out its editor must bypass that path.
func (a *application) showRecordsPanel() {
	a.showRecordPanel(false)
}

func (a *application) popoutRecord() {
	a.showRecordPanel(true)
}

func (a *application) showRecordPanel(preserveRecord bool) {
	if a.window == nil || a.window.IsDestroyed() || a.records.closed {
		return
	}
	if a.recordsPanel == nil {
		a.recordsPanel = mygo.NewWindow(mygo.WindowOptions{
			Title: a.text().Records(), Panel: true, Hidden: true,
			Frameless: true, Transparent: true, Vibrancy: mygo.VibrancyPopover,
			Width: 400, Height: 700, MinWidth: 320, MinHeight: 480,
			Content: ui.View(a.recordsPanelView),
		})
		a.recordsPanel.OnClose(func(e *mygo.CloseEvent) {
			e.PreventDefault()
			a.hideRecordsPanel()
		})
	}
	if !a.recordsPanel.IsVisible() {
		a.cancelRecordWork()
		a.records.detached = true
		a.records.returnToDetail = preserveRecord
		if preserveRecord {
			a.focusRecordInput()
		} else {
			a.openRecords()
		}
		a.loadRecordImages()
	}
	display := mygo.Screen.DisplayNearestPoint(mygo.Screen.CursorScreenPoint())
	area := display.WorkArea
	width, height := a.recordsPanel.Size()
	width, height = min(width, area.Width), min(height, area.Height)
	a.recordsPanel.SetSize(width, height)
	a.recordsPanel.SetPosition(area.X+(area.Width-width)/2, area.Y+(area.Height-height)/3)
	a.recordsPanel.SetTitle(a.text().Records())
	a.recordsPanel.Show()
	a.loadRecords()
	a.invalidateRecords()
}

func (a *application) hideRecordsPanel() {
	a.cancelRecordWork()
	a.records.detached = false
	if a.recordsPanel != nil && a.recordsPanel.IsVisible() {
		a.queueRecords(true)
		a.recordsPanel.Hide()
		a.recordsBrowser.detailOpen = true
		if a.records.returnToDetail {
			a.recordsBrowser.popout.Focus()
		} else {
			a.recordsButton.Focus()
		}
		a.loadRecordImages()
	}
	a.invalidateRecords()
}

func (a *application) destroyRecordsPanel() {
	a.records.closed = true
	a.cancelRecordWork()
	if a.recordsPanel != nil {
		a.recordsPanel.Destroy()
		a.recordsPanel = nil
	}
}

func (a *application) escapeRecordEditor() bool {
	if a.records.previewImage != "" {
		a.closeRecordImagePreview()
		return true
	}
	if a.records.ready && !a.records.review && a.records.book.Active().Edit != nil {
		a.cancelRecordEntryEdit()
		return true
	}
	return false
}

func (a *application) recordsPanelView(c *ui.Context) {
	theme.Apply(c)
	if c.Vibrancy() {
		c.Root().Background(ui.Transparent)
	} else {
		c.Root().Background(c.Theme().Background)
	}
	c.OnShortcut(0, ui.KeyEscape, func() {
		if !a.escapeRecordEditor() {
			a.hideRecordsPanel()
		}
	})
	if a.recordsPanel != nil && !a.records.detached {
		// A hidden native panel can still receive a redraw. It must not bind
		// the same editor state while the journal page owns it.
		ui.Column(c).Fill()
		return
	}
	a.recordsContent(c)
}
