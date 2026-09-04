//go:build windows

// Package secure criptografa segredos (API keys, etc.) usando DPAPI do Windows.
package secure

import (
	"encoding/base64"
	"unsafe"

	"golang.org/x/sys/windows"
)

type dataBlob struct {
	cbData uint32
	pbData *byte
}

func newBlob(d []byte) dataBlob {
	if len(d) == 0 {
		return dataBlob{}
	}
	return dataBlob{cbData: uint32(len(d)), pbData: &d[0]}
}

func (b *dataBlob) bytes() []byte {
	if b.pbData == nil || b.cbData == 0 {
		return nil
	}
	return unsafe.Slice(b.pbData, b.cbData)
}

var (
	modcrypt32             = windows.NewLazySystemDLL("crypt32.dll")
	modkernel32            = windows.NewLazySystemDLL("kernel32.dll")
	procCryptProtectData   = modcrypt32.NewProc("CryptProtectData")
	procCryptUnprotectData = modcrypt32.NewProc("CryptUnprotectData")
	procLocalFree          = modkernel32.NewProc("LocalFree")
)

// CRYPTPROTECT_UI_FORBIDDEN evita qualquer prompt de UI do Windows durante a operação.
const cryptProtectUIForbidden = 0x1

// IsAvailable reporta se a criptografia DPAPI pode ser usada (sempre true no Windows).
func IsAvailable() bool { return true }

// EncryptString criptografa uma string com DPAPI (escopo do usuário atual) e retorna base64.
func EncryptString(plain string) (string, error) {
	if plain == "" {
		return "", nil
	}
	in := newBlob([]byte(plain))
	var out dataBlob
	ret, _, err := procCryptProtectData.Call(
		uintptr(unsafe.Pointer(&in)),
		0, 0, 0, 0,
		cryptProtectUIForbidden,
		uintptr(unsafe.Pointer(&out)),
	)
	if ret == 0 {
		return "", err
	}
	defer procLocalFree.Call(uintptr(unsafe.Pointer(out.pbData)))
	return base64.StdEncoding.EncodeToString(out.bytes()), nil
}

// DecryptString reverte EncryptString.
func DecryptString(encoded string) (string, error) {
	if encoded == "" {
		return "", nil
	}
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}
	in := newBlob(raw)
	var out dataBlob
	ret, _, callErr := procCryptUnprotectData.Call(
		uintptr(unsafe.Pointer(&in)),
		0, 0, 0, 0,
		cryptProtectUIForbidden,
		uintptr(unsafe.Pointer(&out)),
	)
	if ret == 0 {
		return "", callErr
	}
	defer procLocalFree.Call(uintptr(unsafe.Pointer(out.pbData)))
	return string(out.bytes()), nil
}
