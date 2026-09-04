package hub

import (
	"encoding/json"
	"log"
	"strings"
	"sync"

	"github.com/gorilla/websocket"

	"api-chat/internal/chatmsg"
)

type Hub struct {
	mu       sync.Mutex
	clients  map[*websocket.Conn]bool
	botNames map[string]bool
}

func New(botNames map[string]bool) *Hub {
	return &Hub{
		clients:  make(map[*websocket.Conn]bool),
		botNames: botNames,
	}
}

func (h *Hub) Register(conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[conn] = true
}

func (h *Hub) Unregister(conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.clients, conn)
	conn.Close()
}

func (h *Hub) Broadcast(msg chatmsg.Message) {
	if h.botNames[strings.ToLower(msg.Username)] {
		return
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		log.Println("hub: failed to marshal message:", err)
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	for conn := range h.clients {
		if err := conn.WriteMessage(websocket.TextMessage, payload); err != nil {
			conn.Close()
			delete(h.clients, conn)
		}
	}
}
