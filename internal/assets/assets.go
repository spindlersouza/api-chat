// Package assets embute os icones da aplicacao no binario (go:embed).
package assets

import _ "embed"

//go:embed icon.png
var IconPNG []byte

//go:embed logo.png
var LogoPNG []byte
