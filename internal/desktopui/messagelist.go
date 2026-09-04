package desktopui

import (
	"fmt"
	"image/color"
	"strings"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"api-chat/internal/chatmsg"
)

// messageList e uma lista de mensagens (com scroll, sem limite de historico)
// usada tanto pra aba "Chat geral" quanto pra cada aba de plataforma. O
// filtro decide quais mensagens entram nela; nil aceita todas.
type messageList struct {
	list   *widget.List
	filter func(chatmsg.Message) bool

	mu       sync.Mutex
	messages []chatmsg.Message
}

func newMessageList(filter func(chatmsg.Message) bool) *messageList {
	m := &messageList{filter: filter}

	m.list = widget.NewList(
		func() int {
			m.mu.Lock()
			defer m.mu.Unlock()
			return len(m.messages)
		},
		func() fyne.CanvasObject {
			header := canvas.NewText("", color.White)
			header.TextStyle = fyne.TextStyle{Bold: true}
			body := widget.NewLabel("")
			body.Wrapping = fyne.TextWrapWord
			return container.NewVBox(header, body)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			m.mu.Lock()
			if id < 0 || id >= len(m.messages) {
				m.mu.Unlock()
				return
			}
			msg := m.messages[id]
			m.mu.Unlock()

			box := obj.(*fyne.Container)
			header := box.Objects[0].(*canvas.Text)
			body := box.Objects[1].(*widget.Label)

			badges := ""
			if len(msg.Badges) > 0 {
				badges = " [" + strings.Join(msg.Badges, "][") + "]"
			}
			header.Text = fmt.Sprintf("%s%s - %s", msg.Username, badges, strings.ToUpper(msg.Platform))
			header.Color = platformColor(msg.Platform)
			header.Refresh()
			body.SetText(msg.Message)
		},
	)

	return m
}

func platformColor(platform string) color.Color {
	switch platform {
	case "twitch":
		return color.RGBA{R: 0x91, G: 0x46, B: 0xFF, A: 0xFF}
	case "youtube":
		return color.RGBA{R: 0xFF, G: 0x00, B: 0x00, A: 0xFF}
	case "kick":
		return color.RGBA{R: 0x53, G: 0xFC, B: 0x18, A: 0xFF}
	default:
		return color.White
	}
}

// maybeAdd adiciona a mensagem se ela passar pelo filtro. Seguro pra chamar
// de qualquer goroutine.
func (m *messageList) maybeAdd(msg chatmsg.Message) {
	if m.filter != nil && !m.filter(msg) {
		return
	}

	m.mu.Lock()
	m.messages = append(m.messages, msg)
	m.mu.Unlock()

	fyne.Do(func() {
		m.list.Refresh()
		m.list.ScrollToBottom()
	})
}

func (m *messageList) content() fyne.CanvasObject {
	return m.list
}
