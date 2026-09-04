package desktopui

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"

	"fyne.io/fyne/v2"

	"api-chat/internal/assets"
)

// loadTrayIcons decodes the embedded app icon and derives a second version
// with a red "unread" dot badged in the top-right corner.
func loadTrayIcons() (normal, unread fyne.Resource, err error) {
	img, err := png.Decode(bytes.NewReader(assets.IconPNG))
	if err != nil {
		return nil, nil, err
	}

	normal = fyne.NewStaticResource("tray.png", assets.IconPNG)

	var buf bytes.Buffer
	if err := png.Encode(&buf, badgeWithDot(img)); err != nil {
		return nil, nil, err
	}
	unread = fyne.NewStaticResource("tray-unread.png", buf.Bytes())

	return normal, unread, nil
}

func badgeWithDot(src image.Image) image.Image {
	b := src.Bounds()
	dst := image.NewRGBA(b)
	draw.Draw(dst, b, src, b.Min, draw.Src)

	size := b.Dx()
	radius := size / 6
	if radius < 6 {
		radius = 6
	}
	cx := b.Max.X - radius - size/14
	cy := b.Min.Y + radius + size/14

	red := color.RGBA{R: 230, G: 40, B: 40, A: 255}
	border := color.RGBA{R: 20, G: 20, B: 20, A: 255}

	for y := cy - radius - 2; y <= cy+radius+2; y++ {
		for x := cx - radius - 2; x <= cx+radius+2; x++ {
			dx, dy := x-cx, y-cy
			dist2 := dx*dx + dy*dy
			switch {
			case dist2 <= radius*radius:
				dst.Set(x, y, red)
			case dist2 <= (radius+2)*(radius+2):
				dst.Set(x, y, border)
			}
		}
	}

	return dst
}
