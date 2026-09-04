package desktopui

import (
	"image/color"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"api-chat/internal/config"
)

func newConfigView(cfg *config.Config, startedAlready bool, onSaved func()) fyne.CanvasObject {
	newEntry := func() *widget.Entry {
		e := widget.NewEntry()
		e.Wrapping = fyne.TextWrapOff
		e.Scroll = fyne.ScrollNone
		return e
	}

	newPasswordEntry := func() *widget.Entry {
		e := widget.NewPasswordEntry()
		e.Wrapping = fyne.TextWrapOff
		e.Scroll = fyne.ScrollNone
		return e
	}

	twitchEntry := newEntry()
	twitchEntry.SetText(cfg.TwitchChannel)

	ytKeyEntry := newPasswordEntry()
	ytKeyEntry.SetText(cfg.YouTubeAPIKey)

	ytChannelEntry := newEntry()
	ytChannelEntry.SetText(cfg.YouTubeChannelID)

	pollEntry := newEntry()
	pollEntry.SetText(strconv.Itoa(cfg.PollMinIntervalMs / 1000))

	kickChannelEntry := newEntry()
	kickChannelEntry.SetText(cfg.KickChannel)

	kickChatroomEntry := newEntry()
	kickChatroomEntry.SetText(cfg.KickChatroomID)

	botsEntry := newEntry()
	botsEntry.SetText(cfg.BotUsernamesText())

	cooldownEntry := newEntry()
	cooldownEntry.SetText(strconv.Itoa(cfg.AlertCooldownMs / 1000))

	portEntry := newEntry()
	portEntry.SetText(cfg.Port)

	statusLabel := widget.NewLabel("")
	statusLabel.Wrapping = fyne.TextWrapWord

	save := func() {
		cfg.TwitchChannel = strings.ToLower(strings.TrimSpace(twitchEntry.Text))
		cfg.YouTubeAPIKey = strings.TrimSpace(ytKeyEntry.Text)
		cfg.YouTubeChannelID = strings.TrimSpace(ytChannelEntry.Text)
		cfg.KickChannel = strings.ToLower(strings.TrimSpace(kickChannelEntry.Text))
		cfg.KickChatroomID = strings.TrimSpace(kickChatroomEntry.Text)
		cfg.SetBotUsernamesText(botsEntry.Text)

		if v, err := strconv.Atoi(strings.TrimSpace(pollEntry.Text)); err == nil && v > 0 {
			cfg.PollMinIntervalMs = v * 1000
		}

		if v, err := strconv.Atoi(strings.TrimSpace(cooldownEntry.Text)); err == nil && v > 0 {
			cfg.AlertCooldownMs = v * 1000
		}

		if p := strings.TrimSpace(portEntry.Text); p != "" {
			cfg.Port = p
		}

		if err := cfg.Save(); err != nil {
			statusLabel.SetText("Erro ao salvar: " + err.Error())
			return
		}

		statusLabel.SetText("")
		onSaved()
	}

	saveBtn := widget.NewButtonWithIcon("Salvar", theme.ConfirmIcon(), save)
	saveBtn.Importance = widget.HighImportance

	field := func(label string, entry *widget.Entry) fyne.CanvasObject {
		text := widget.NewLabelWithStyle(label, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
		text.Importance = widget.LowImportance
		return container.NewVBox(text, entry)
	}

	items := []fyne.CanvasObject{
		widget.NewLabelWithStyle("Configuracao", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
	}

	if startedAlready {
		notice := widget.NewLabel("Os servicos ja estao rodando: mudancas so valem depois de reiniciar o app.")
		notice.Wrapping = fyne.TextWrapWord
		items = append(items, notice)
	}

	items = append(items,
		platformSection("twitch", "Twitch",
			field("Canal", twitchEntry),
		),
		platformSection("youtube", "YouTube",
			field("API Key", ytKeyEntry),
			field("Channel ID", ytChannelEntry),
			field("Intervalo de verificacao do chat (segundos)", pollEntry),
		),
		platformSection("kick", "Kick",
			field("Canal", kickChannelEntry),
			field("Chatroom ID (opcional, preencher so se a busca automatica falhar)", kickChatroomEntry),
		),
		neutralSection("Geral",
			field("Bots a ignorar (separados por virgula)", botsEntry),
			field("Cooldown do alerta na bandeja (segundos)", cooldownEntry),
			field("Porta preferida (tenta as seguintes se estiver ocupada)", portEntry),
		),
		container.NewCenter(saveBtn),
		statusLabel,
	)

	return container.NewVScroll(container.NewVBox(items...))
}

// platformSection agrupa os campos de uma rede num cartao com um fundo suave
// e uma borda na cor da marca, pra ficar facil de identificar visualmente.
func platformSection(platform, title string, fields ...fyne.CanvasObject) fyne.CanvasObject {
	brand := platformColor(platform)
	return sectionCard(title, brand, fields...)
}

func neutralSection(title string, fields ...fyne.CanvasObject) fyne.CanvasObject {
	return sectionCard(title, color.Gray{Y: 0x88}, fields...)
}

func sectionCard(title string, brand color.Color, fields ...fyne.CanvasObject) fyne.CanvasObject {
	r, g, b, _ := brand.RGBA()
	bg := color.NRGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: 28}

	bgRect := canvas.NewRectangle(bg)
	bgRect.StrokeColor = brand
	bgRect.StrokeWidth = 1
	bgRect.CornerRadius = 8

	heading := widget.NewLabelWithStyle(title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	body := container.NewVBox(append([]fyne.CanvasObject{heading}, fields...)...)
	padded := container.NewPadded(body)

	return container.NewStack(bgRect, padded)
}
