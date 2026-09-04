package desktopui

import (
	"bytes"
	"image"
	"image/png"

	"fyne.io/fyne/v2"
)

// platformDotIcon gera um circulo solido na cor da rede, usado como icone da
// aba pra cada plataforma ficar visualmente identificavel.
func platformDotIcon(platform string) fyne.Resource {
	const size = 32
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	c := platformColor(platform)
	cx, cy, r := size/2, size/2, size/2-2

	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dx, dy := x-cx, y-cy
			if dx*dx+dy*dy <= r*r {
				img.Set(x, y, c)
			}
		}
	}

	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return fyne.NewStaticResource(platform+"-dot.png", buf.Bytes())
}
