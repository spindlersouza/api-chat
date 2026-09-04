package server

import (
	"embed"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"

	"github.com/gorilla/websocket"

	"api-chat/internal/hub"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Server struct {
	hub   *hub.Hub
	webFS embed.FS
	port  string
}

func New(h *hub.Hub, webFS embed.FS, port string) *Server {
	return &Server{hub: h, webFS: webFS, port: port}
}

// Listen tenta a porta preferida e, se estiver ocupada, vai tentando as
// proximas (ate 50 tentativas). Retorna o listener ja aberto e a porta
// efetivamente usada, pra quem chamou saber a URL real do servidor.
func (s *Server) Listen() (net.Listener, string, error) {
	start, err := strconv.Atoi(s.port)
	if err != nil || start <= 0 {
		start = 3000
	}

	for p := start; p < start+50; p++ {
		addr := ":" + strconv.Itoa(p)
		ln, err := net.Listen("tcp", addr)
		if err == nil {
			if p != start {
				log.Printf("server: porta %d ocupada, usando %d\n", start, p)
			}
			port := strconv.Itoa(p)
			s.port = port
			return ln, port, nil
		}
	}

	return nil, "", fmt.Errorf("nenhuma porta livre entre %d e %d", start, start+49)
}

// Serve sobe o servidor HTTP no listener ja aberto (bloqueia ate encerrar).
func (s *Server) Serve(ln net.Listener) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/overlay", s.handleFile("web/overlay.html", "text/html; charset=utf-8"))
	mux.HandleFunc("/overlay.css", s.handleFile("web/overlay.css", "text/css; charset=utf-8"))
	mux.HandleFunc("/monitor", s.handleFile("web/monitor.html", "text/html; charset=utf-8"))
	mux.HandleFunc("/monitor.css", s.handleFile("web/monitor.css", "text/css; charset=utf-8"))
	mux.HandleFunc("/ws", s.handleWS)

	log.Printf("server: listening on http://localhost:%s/overlay\n", s.port)
	return http.Serve(ln, mux)
}

// handleFile serves a file from disk if present (allows easy customization
// without recompiling), falling back to the embedded default otherwise.
func (s *Server) handleFile(path, contentType string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", contentType)

		if data, err := os.ReadFile(path); err == nil {
			w.Write(data)
			return
		}

		data, err := s.webFS.ReadFile(path)
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Write(data)
	}
}

func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("server: websocket upgrade failed:", err)
		return
	}

	s.hub.Register(conn)
	defer s.hub.Unregister(conn)

	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}
}
