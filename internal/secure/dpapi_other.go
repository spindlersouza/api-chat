//go:build !windows

// Fallback para plataformas sem DPAPI: retorna o valor em texto claro.
package secure

func IsAvailable() bool { return false }

func EncryptString(plain string) (string, error) { return plain, nil }

func DecryptString(encoded string) (string, error) { return encoded, nil }
