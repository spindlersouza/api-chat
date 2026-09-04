package desktopui

import (
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"

	"api-chat/internal/logbuf"
)

// logView mostra em tempo real o que vai pro log (mesmo conteudo que antes ia
// pro console).
type logView struct {
	list *widget.List

	mu    sync.Mutex
	lines []string
}

func newLogView(logs *logbuf.Buffer) *logView {
	v := &logView{lines: logs.Snapshot()}

	v.list = widget.NewList(
		func() int {
			v.mu.Lock()
			defer v.mu.Unlock()
			return len(v.lines)
		},
		func() fyne.CanvasObject {
			label := widget.NewLabel("")
			label.Wrapping = fyne.TextWrapWord
			return label
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			v.mu.Lock()
			text := ""
			if id >= 0 && id < len(v.lines) {
				text = v.lines[id]
			}
			v.mu.Unlock()
			obj.(*widget.Label).SetText(text)
		},
	)

	logs.SetOnAppend(func(line string) {
		v.mu.Lock()
		v.lines = append(v.lines, line)
		v.mu.Unlock()
		fyne.Do(func() {
			v.list.Refresh()
			v.list.ScrollToBottom()
		})
	})

	return v
}

func (v *logView) content() fyne.CanvasObject {
	return v.list
}
