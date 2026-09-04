package kick

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"time"

	"github.com/gorilla/websocket"

	"api-chat/internal/chatmsg"
)

// Public Pusher app key used by kick.com's own web client to broadcast chat.
const pusherURL = "wss://ws-us2.pusher.com/app/32cbd69e4b950bf97679?protocol=7&client=js&version=8.4.0&flash=false"

const kickAPIBase = "https://kick.com/api/v2/channels/"

var emoteRe = regexp.MustCompile(`\[emote:(\d+):([^\]]+)\]`)

// Run connects to Kick's chat (via its public Pusher websocket) for the
// given channel slug and streams messages into onMessage. It reconnects
// automatically on any error. If chatroomID is already known it skips the
// channel-lookup HTTP call (useful when that endpoint is blocked).
func Run(channelSlug, chatroomID string, onMessage func(chatmsg.Message)) {
	if channelSlug == "" && chatroomID == "" {
		log.Println("kick: KICK_CHANNEL not set, skipping")
		return
	}

	backoff := time.Second
	for {
		err := connectAndListen(channelSlug, chatroomID, onMessage)
		if err != nil {
			log.Println("kick: connection error:", err)
		}
		log.Printf("kick: reconnecting in %s...\n", backoff)
		time.Sleep(backoff)
		if backoff < 30*time.Second {
			backoff *= 2
		}
	}
}

func connectAndListen(channelSlug, chatroomID string, onMessage func(chatmsg.Message)) error {
	id := chatroomID
	if id == "" {
		var err error
		id, err = fetchChatroomID(channelSlug)
		if err != nil {
			return fmt.Errorf("discovering chatroom id: %w", err)
		}
	}

	conn, _, err := websocket.DefaultDialer.Dial(pusherURL, nil)
	if err != nil {
		return err
	}
	defer conn.Close()

	sub := map[string]interface{}{
		"event": "pusher:subscribe",
		"data":  map[string]string{"channel": "chatrooms." + id + ".v2"},
	}
	if err := conn.WriteJSON(sub); err != nil {
		return err
	}

	log.Printf("kick: connected, joined chatroom %s\n", id)

	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			return err
		}

		var envelope struct {
			Event string `json:"event"`
			Data  string `json:"data"`
		}
		if err := json.Unmarshal(raw, &envelope); err != nil {
			continue
		}

		switch envelope.Event {
		case "pusher:ping":
			conn.WriteJSON(map[string]interface{}{"event": "pusher:pong", "data": map[string]string{}})
		case "App\\Events\\ChatMessageEvent":
			if msg, ok := parseChatMessage(envelope.Data); ok {
				onMessage(msg)
			}
		}
	}
}

func fetchChatroomID(slug string) (string, error) {
	req, err := http.NewRequest("GET", kickAPIBase+url.PathEscape(slug), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("kick api error %d: %s", resp.StatusCode, string(body))
	}

	var data struct {
		Chatroom struct {
			ID int `json:"id"`
		} `json:"chatroom"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		return "", err
	}
	if data.Chatroom.ID == 0 {
		return "", fmt.Errorf("chatroom id not found in response")
	}
	return strconv.Itoa(data.Chatroom.ID), nil
}

type kickMessage struct {
	Content string `json:"content"`
	Sender  struct {
		Username string `json:"username"`
		Identity struct {
			Color  string `json:"color"`
			Badges []struct {
				Type string `json:"type"`
			} `json:"badges"`
		} `json:"identity"`
	} `json:"sender"`
}

func parseChatMessage(data string) (chatmsg.Message, bool) {
	var km kickMessage
	if err := json.Unmarshal([]byte(data), &km); err != nil {
		return chatmsg.Message{}, false
	}

	text, emotes := parseEmotes(km.Content)

	badges := []string{}
	for _, b := range km.Sender.Identity.Badges {
		if b.Type != "" {
			badges = append(badges, b.Type)
		}
	}

	return chatmsg.Message{
		Platform:  "kick",
		Username:  km.Sender.Username,
		Message:   text,
		Avatar:    "",
		Badges:    badges,
		Color:     km.Sender.Identity.Color,
		Emotes:    emotes,
		Timestamp: time.Now().UnixMilli(),
	}, true
}

// parseEmotes replaces Kick's inline `[emote:ID:Name]` tokens with the emote
// name and collects the corresponding emote image references.
func parseEmotes(content string) (string, []chatmsg.Emote) {
	emotes := []chatmsg.Emote{}
	text := emoteRe.ReplaceAllStringFunc(content, func(m string) string {
		groups := emoteRe.FindStringSubmatch(m)
		id, name := groups[1], groups[2]
		emotes = append(emotes, chatmsg.Emote{
			Code: name,
			URL:  fmt.Sprintf("https://files.kick.com/emotes/%s/fullsize", id),
		})
		return name
	})
	return text, emotes
}
