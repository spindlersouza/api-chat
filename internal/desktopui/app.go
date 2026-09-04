package desktopui

import (
	"image/color"
	"log"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"api-chat/internal/assets"
	"api-chat/internal/chatmsg"
	"api-chat/internal/config"
	"api-chat/internal/logbuf"
)

// AppName e o nome exibido em titulos, menu da bandeja e notificacoes.
const AppName = "TraitAPI - Multichat"

// ReadyFunc e chamada uma unica vez, depois do login (e da config, se for a
// primeira vez), pra efetivamente subir hub/conexoes/servidor HTTP. Retorna a
// URL da overlay ja com a porta real usada.
type ReadyFunc func(cfg *config.Config) (overlayURL string, err error)

// App orquestra as telas da janela nativa: login -> config (so na primeira
// vez) -> abas (chat geral, uma por rede configurada, log).
type App struct {
	fyneApp fyne.App
	win     fyne.Window
	cfg     *config.Config
	logs    *logbuf.Buffer
	ready   ReadyFunc

	mu         sync.Mutex
	visible    bool
	started    bool
	overlayURL string
	username   string

	geral   *messageList
	byPlat  map[string]*messageList
	pending []chatmsg.Message

	onVisibilityChange func(visible bool)
}

// windowSize e forcado de novo a cada troca de tela: sem isso o Fyne encolhe
// a janela pro tamanho minimo do conteudo atual, e a rolagem da tela de
// config fica "presa" num viewport menor que o formulario.
var windowSize = fyne.NewSize(600, 720)

func NewApp(a fyne.App, win fyne.Window, cfg *config.Config, logs *logbuf.Buffer, ready ReadyFunc) *App {
	app := &App{fyneApp: a, win: win, cfg: cfg, logs: logs, ready: ready, byPlat: map[string]*messageList{}}

	win.Resize(windowSize)
	win.SetCloseIntercept(func() {
		win.Hide()
		app.setVisible(false)
	})

	app.showLogin()
	return app
}

func (app *App) setContent(content fyne.CanvasObject) {
	// Resize ANTES do SetContent: se o resize acontece depois, o Scroll
	// interno faz layout do conteudo contra um tamanho intermediario e o
	// offset maximo calculado fica errado (rolagem "perde" o progresso).
	app.win.Resize(windowSize)
	app.win.SetContent(content)
}

func (app *App) showLogin() {
	app.setContent(newLoginView(app.cfg, func(username string) {
		app.username = username
		if !config.Exists() {
			app.showConfig(false)
			return
		}
		app.ensureStarted()
	}))
}

func (app *App) showConfig(startedAlready bool) {
	configContent := newConfigView(app.cfg, startedAlready, func() {
		if startedAlready {
			app.showMain()
			return
		}
		app.ensureStarted()
	})
	root := container.NewBorder(app.buildTopBar(false), nil, nil, nil, configContent)
	app.setContent(root)
}

func (app *App) ensureStarted() {
	app.mu.Lock()
	if app.started {
		app.mu.Unlock()
		app.showMain()
		return
	}
	app.mu.Unlock()

	url, err := app.ready(app.cfg)
	if err != nil {
		log.Println("app: failed to start services:", err)
	}

	app.mu.Lock()
	app.overlayURL = url
	app.started = true
	app.mu.Unlock()

	app.buildMessageLists()
	app.flushPending()
	app.showMain()
}

func (app *App) buildMessageLists() {
	app.geral = newMessageList(nil)
	if app.cfg.TwitchChannel != "" {
		app.byPlat["twitch"] = newMessageList(platformFilter("twitch"))
	}
	if app.cfg.YouTubeChannelID != "" && app.cfg.YouTubeAPIKey != "" {
		app.byPlat["youtube"] = newMessageList(platformFilter("youtube"))
	}
	if app.cfg.KickChannel != "" || app.cfg.KickChatroomID != "" {
		app.byPlat["kick"] = newMessageList(platformFilter("kick"))
	}
}

func platformFilter(platform string) func(chatmsg.Message) bool {
	return func(m chatmsg.Message) bool { return m.Platform == platform }
}

func (app *App) flushPending() {
	app.mu.Lock()
	pending := app.pending
	app.pending = nil
	app.mu.Unlock()

	for _, msg := range pending {
		app.dispatch(msg)
	}
}

func (app *App) showMain() {
	tabs := container.NewAppTabs(container.NewTabItem("Chat geral", app.geral.content()))
	if v, ok := app.byPlat["twitch"]; ok {
		tabs.Append(container.NewTabItemWithIcon("Twitch", platformDotIcon("twitch"), v.content()))
	}
	if v, ok := app.byPlat["youtube"]; ok {
		tabs.Append(container.NewTabItemWithIcon("YouTube", platformDotIcon("youtube"), v.content()))
	}
	if v, ok := app.byPlat["kick"]; ok {
		tabs.Append(container.NewTabItemWithIcon("Kick", platformDotIcon("kick"), v.content()))
	}
	tabs.Append(container.NewTabItem("Log", newLogView(app.logs).content()))

	root := container.NewBorder(app.buildTopBar(true), nil, nil, nil, tabs)
	app.setContent(root)
}

// buildTopBar monta o cabeçalho compacto acima das abas: marca e usuario
// logado sempre; a linha da URL da overlay (com copiar/engrenagem) so
// aparece quando showURL e true (ela nao faz sentido na propria tela de
// config, que ainda nao tem uma URL definitiva na primeira vez).
func (app *App) buildTopBar(showURL bool) fyne.CanvasObject {
	logoImg := canvas.NewImageFromResource(fyne.NewStaticResource("logo-small.png", assets.LogoPNG))
	logoImg.FillMode = canvas.ImageFillContain
	logoImg.SetMinSize(fyne.NewSize(22, 22))

	brand := canvas.NewText("Multichat", color.White)
	brand.TextStyle = fyne.TextStyle{Bold: true}
	brand.TextSize = 14
	row1 := container.NewHBox(logoImg, brand)

	userText := canvas.NewText("Usuario: "+app.username, color.Gray{Y: 0xcc})
	userText.TextSize = 12

	if !showURL {
		return container.NewVBox(row1, userText, widget.NewSeparator())
	}

	app.mu.Lock()
	url := app.overlayURL
	app.mu.Unlock()

	urlLabel := widget.NewLabel(url)
	copyBtn := widget.NewButtonWithIcon("", theme.ContentCopyIcon(), func() {
		app.win.Clipboard().SetContent(url)
	})
	gearBtn := widget.NewButtonWithIcon("", theme.SettingsIcon(), func() {
		app.showConfig(true)
	})
	row3 := container.NewBorder(nil, nil, nil, container.NewHBox(copyBtn, gearBtn), urlLabel)

	return container.NewVBox(row1, userText, row3, widget.NewSeparator())
}

// PushMessage e chamado (de qualquer goroutine) a cada mensagem nova do chat.
func (app *App) PushMessage(msg chatmsg.Message) {
	app.mu.Lock()
	if app.geral == nil {
		app.pending = append(app.pending, msg)
		app.mu.Unlock()
		return
	}
	app.mu.Unlock()

	app.dispatch(msg)
}

func (app *App) dispatch(msg chatmsg.Message) {
	app.geral.maybeAdd(msg)
	if v, ok := app.byPlat[msg.Platform]; ok {
		v.maybeAdd(msg)
	}
}

// Show exibe a janela e marca como visivel (limpa o estado de "nao lido").
func (app *App) Show() {
	app.win.Show()
	app.win.RequestFocus()
	app.setVisible(true)
}

func (app *App) Visible() bool {
	app.mu.Lock()
	defer app.mu.Unlock()
	return app.visible
}

func (app *App) setVisible(v bool) {
	app.mu.Lock()
	app.visible = v
	cb := app.onVisibilityChange
	app.mu.Unlock()
	if cb != nil {
		cb(v)
	}
}

// OnVisibilityChange registra o callback disparado quando a janela e aberta
// ou escondida.
func (app *App) OnVisibilityChange(cb func(visible bool)) {
	app.onVisibilityChange = cb
}
