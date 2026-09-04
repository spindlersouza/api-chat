// Package logbuf guarda as ultimas linhas de log em memoria pra alimentar a
// aba "Log" da janela nativa, alem do arquivo de log em disco.
package logbuf

import (
	"strings"
	"sync"
)

const maxLines = 2000

type Buffer struct {
	mu       sync.Mutex
	lines    []string
	onAppend func(line string)
}

func New() *Buffer {
	return &Buffer{}
}

// Write implementa io.Writer, pra ser usado com log.SetOutput /
// io.MultiWriter junto do arquivo de log.
func (b *Buffer) Write(p []byte) (int, error) {
	text := strings.TrimRight(string(p), "\n")
	if text == "" {
		return len(p), nil
	}

	for _, line := range strings.Split(text, "\n") {
		b.append(line)
	}
	return len(p), nil
}

func (b *Buffer) append(line string) {
	b.mu.Lock()
	b.lines = append(b.lines, line)
	if len(b.lines) > maxLines {
		b.lines = b.lines[len(b.lines)-maxLines:]
	}
	cb := b.onAppend
	b.mu.Unlock()

	if cb != nil {
		cb(line)
	}
}

// Snapshot retorna uma copia das linhas atuais.
func (b *Buffer) Snapshot() []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]string, len(b.lines))
	copy(out, b.lines)
	return out
}

// SetOnAppend registra o callback chamado a cada linha nova.
func (b *Buffer) SetOnAppend(cb func(line string)) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.onAppend = cb
}
