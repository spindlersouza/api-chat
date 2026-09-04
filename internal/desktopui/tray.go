// Package desktopui monta o icone da bandeja e a janela nativa de leitura do
// chat, pra nao precisar deixar o navegador aberto pra acompanhar as
// mensagens.
package desktopui

import (
	"log"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"

	"api-chat/internal/assets"
	"api-chat/internal/chatmsg"
)

// Tray liga o icone da bandeja, o menu e o alerta com temporizador: a
// primeira mensagem de uma leva "silenciosa" dispara uma notificacao e marca
// o icone com um ponto vermelho; enquanto o cooldown nao passar, mensagens
// novas so atualizam o ponto, sem notificacao repetida.
type Tray struct {
	app      fyne.App
	desk     desktop.App
	window   *App
	cooldown time.Duration

	normalIcon fyne.Resource
	unreadIcon fyne.Resource

	mu        sync.Mutex
	unread    bool
	lastAlert time.Time
}

func Setup(a fyne.App, window *App, cooldown time.Duration) *Tray {
	t := &Tray{app: a, window: window, cooldown: cooldown}

	normal, unread, err := loadTrayIcons()
	if err != nil {
		log.Println("desktopui: failed to build tray icons:", err)
		normal = fyne.NewStaticResource("tray.png", assets.IconPNG)
		unread = normal
	}
	t.normalIcon, t.unreadIcon = normal, unread

	if desk, ok := a.(desktop.App); ok {
		t.desk = desk
		a.Lifecycle().SetOnStarted(func() {
			desk.SetSystemTrayIcon(t.normalIcon)
			t.rebuildMenu()
		})
	}

	window.OnVisibilityChange(func(visible bool) {
		if visible {
			t.clearUnread()
		}
	})

	return t
}

func (t *Tray) rebuildMenu() {
	if t.desk == nil {
		return
	}

	items := []*fyne.MenuItem{
		fyne.NewMenuItem("Abrir", func() { t.window.Show() }),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Sair", func() { t.app.Quit() }),
	}
	t.desk.SetSystemTrayMenu(fyne.NewMenu(AppName, items...))
}

// OnMessage e chamado (de qualquer goroutine) a cada mensagem nova do chat.
func (t *Tray) OnMessage(msg chatmsg.Message) {
	t.window.PushMessage(msg)

	if t.window.Visible() {
		return
	}

	t.mu.Lock()
	shouldNotify := time.Since(t.lastAlert) >= t.cooldown
	wasUnread := t.unread
	t.unread = true
	if shouldNotify {
		t.lastAlert = time.Now()
	}
	t.mu.Unlock()

	if !wasUnread {
		fyne.Do(func() {
			if t.desk != nil {
				t.desk.SetSystemTrayIcon(t.unreadIcon)
			}
		})
	}

	if shouldNotify {
		t.app.SendNotification(fyne.NewNotification(AppName, "Novas mensagens no chat"))
	}
}

func (t *Tray) clearUnread() {
	t.mu.Lock()
	t.unread = false
	t.mu.Unlock()

	fyne.Do(func() {
		if t.desk != nil {
			t.desk.SetSystemTrayIcon(t.normalIcon)
		}
	})
}
