package main

import (
	"embed"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"fyne.io/fyne/v2"
	fyneapp "fyne.io/fyne/v2/app"

	"api-chat/internal/assets"
	"api-chat/internal/chatmsg"
	"api-chat/internal/config"
	"api-chat/internal/desktopui"
	"api-chat/internal/hub"
	"api-chat/internal/kick"
	"api-chat/internal/logbuf"
	"api-chat/internal/server"
	"api-chat/internal/twitch"
	"api-chat/internal/youtube"
)

//go:embed web/*
var webFS embed.FS

// setupLogging redireciona os logs pro arquivo (temporario: comeca vazio a
// cada execucao e e apagado ao fechar) e pro buffer em memoria usado pela
// aba "Log" da janela nativa.
func setupLogging(logs *logbuf.Buffer) (cleanup func()) {
	logPath := "api-chat.log"
	if exePath, err := os.Executable(); err == nil {
		logPath = filepath.Join(filepath.Dir(exePath), "api-chat.log")
	}

	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		log.SetOutput(logs)
		return func() {}
	}

	log.SetOutput(io.MultiWriter(f, logs))
	return func() {
		f.Close()
		os.Remove(logPath)
	}
}

func main() {
	logs := logbuf.New()
	cleanupLog := setupLogging(logs)
	defer cleanupLog()

	cfg := config.Load()

	fyneApp := fyneapp.NewWithID("com.traitbr.apichat")
	fyneApp.SetIcon(fyne.NewStaticResource("icon.png", assets.IconPNG))
	win := fyneApp.NewWindow(desktopui.AppName)

	var tray *desktopui.Tray

	ready := func(cfg *config.Config) (string, error) {
		h := hub.New(cfg.BotUsernamesMap())

		onMessage := func(msg chatmsg.Message) {
			h.Broadcast(msg)
			if tray != nil {
				tray.OnMessage(msg)
			}
		}

		go twitch.Run(cfg.TwitchChannel, onMessage)
		go youtube.Run(cfg.YouTubeAPIKey, cfg.YouTubeChannelID, cfg.PollMinIntervalMs, onMessage)
		go kick.Run(cfg.KickChannel, cfg.KickChatroomID, onMessage)

		srv := server.New(h, webFS, cfg.Port)
		ln, port, err := srv.Listen()
		if err != nil {
			return "", err
		}
		go func() {
			if err := srv.Serve(ln); err != nil {
				log.Println("server:", err)
			}
		}()

		return fmt.Sprintf("http://localhost:%s/overlay", port), nil
	}

	app := desktopui.NewApp(fyneApp, win, cfg, logs, ready)
	tray = desktopui.Setup(fyneApp, app, time.Duration(cfg.AlertCooldownMs)*time.Millisecond)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Println("encerrando...")
		fyneApp.Quit()
	}()

	win.Show()
	fyneApp.Run()
}
