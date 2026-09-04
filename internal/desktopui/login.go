package desktopui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"api-chat/internal/assets"
	"api-chat/internal/config"
)

// newLoginView monta a tela inicial de login (usuario/senha, admin/admin por
// padrao). O app so segue adiante depois de autenticar.
func newLoginView(cfg *config.Config, onSuccess func(username string)) fyne.CanvasObject {
	logoImg := canvas.NewImageFromResource(fyne.NewStaticResource("logo.png", assets.LogoPNG))
	logoImg.FillMode = canvas.ImageFillContain
	logoImg.SetMinSize(fyne.NewSize(110, 110))

	subtitle := canvas.NewText("Multichat", color.White)
	subtitle.TextStyle = fyne.TextStyle{Bold: true}
	subtitle.TextSize = 20
	subtitle.Alignment = fyne.TextAlignCenter

	fieldSize := fyne.NewSize(windowSize.Width*0.5, 36)

	userEntry := widget.NewEntry()
	userEntry.SetPlaceHolder("Usuario")

	passEntry := widget.NewPasswordEntry()
	passEntry.SetPlaceHolder("Senha")

	errLabel := widget.NewLabel("")
	errLabel.Wrapping = fyne.TextWrapWord
	errLabel.Alignment = fyne.TextAlignCenter

	attempt := func() {
		if cfg.CheckPassword(userEntry.Text, passEntry.Text) {
			errLabel.SetText("")
			onSuccess(userEntry.Text)
			return
		}
		errLabel.SetText("Usuario ou senha invalidos.")
	}
	passEntry.OnSubmitted = func(string) { attempt() }

	loginBtn := widget.NewButtonWithIcon("Entrar", theme.LoginIcon(), attempt)
	loginBtn.Importance = widget.HighImportance

	form := container.NewVBox(
		container.NewCenter(logoImg),
		container.NewCenter(subtitle),
		widget.NewSeparator(),
		container.NewCenter(container.NewGridWrap(fieldSize, userEntry)),
		container.NewCenter(container.NewGridWrap(fieldSize, passEntry)),
		container.NewCenter(loginBtn),
		container.NewCenter(errLabel),
	)

	return container.NewCenter(form)
}
