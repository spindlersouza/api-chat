package twitch

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"api-chat/internal/chatmsg"
)

const ircAddr = "irc.chat.twitch.tv:6697"

// Run connects to Twitch IRC anonymously and streams chat messages for
// `channel` into onMessage. It reconnects automatically on any error.
func Run(channel string, onMessage func(chatmsg.Message)) {
	if channel == "" {
		log.Println("twitch: TWITCH_CHANNEL not set, skipping")
		return
	}

	backoff := time.Second
	for {
		err := connectAndListen(channel, onMessage)
		if err != nil {
			log.Println("twitch: connection error:", err)
		}
		log.Printf("twitch: reconnecting in %s...\n", backoff)
		time.Sleep(backoff)
		if backoff < 30*time.Second {
			backoff *= 2
		}
	}
}

func connectAndListen(channel string, onMessage func(chatmsg.Message)) error {
	conn, err := tls.Dial("tcp", ircAddr, &tls.Config{})
	if err != nil {
		return err
	}
	defer conn.Close()

	nick := fmt.Sprintf("justinfan%d", rand.Intn(90000)+10000)

	fmt.Fprintf(conn, "CAP REQ :twitch.tv/tags twitch.tv/commands twitch.tv/membership\r\n")
	fmt.Fprintf(conn, "PASS blah\r\n")
	fmt.Fprintf(conn, "NICK %s\r\n", nick)
	fmt.Fprintf(conn, "JOIN #%s\r\n", strings.ToLower(channel))

	log.Printf("twitch: connected, joined #%s\n", channel)

	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 0, 64*1024), 64*1024)

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "PING") {
			fmt.Fprintf(conn, "PONG :tmi.twitch.tv\r\n")
			continue
		}

		if msg, ok := parsePrivmsg(line); ok {
			onMessage(msg)
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}
	return fmt.Errorf("connection closed by server")
}

// parsePrivmsg parses an IRC line like:
// @badges=broadcaster/1;color=#FF0000;display-name=Foo;emotes=25:0-4 :foo!foo@foo.tmi.twitch.tv PRIVMSG #channel :hello world
func parsePrivmsg(line string) (chatmsg.Message, bool) {
	var tags map[string]string
	rest := line

	if strings.HasPrefix(line, "@") {
		parts := strings.SplitN(line, " ", 2)
		if len(parts) != 2 {
			return chatmsg.Message{}, false
		}
		tags = parseTags(parts[0][1:])
		rest = parts[1]
	}

	if !strings.Contains(rest, " PRIVMSG #") {
		return chatmsg.Message{}, false
	}

	idx := strings.Index(rest, " :")
	if idx == -1 {
		return chatmsg.Message{}, false
	}
	text := rest[idx+2:]

	username := tags["display-name"]
	if username == "" {
		// fallback: extract from prefix "nick!user@host"
		if strings.HasPrefix(rest, ":") {
			bang := strings.Index(rest, "!")
			if bang > 0 {
				username = rest[1:bang]
			}
		}
	}

	badges := []string{}
	if raw := tags["badges"]; raw != "" {
		for _, b := range strings.Split(raw, ",") {
			name := strings.SplitN(b, "/", 2)[0]
			if name != "" {
				badges = append(badges, name)
			}
		}
	}

	emotes := parseEmotes(tags["emotes"], text)

	return chatmsg.Message{
		Platform:  "twitch",
		Username:  username,
		Message:   text,
		Avatar:    "",
		Badges:    badges,
		Color:     tags["color"],
		Emotes:    emotes,
		Timestamp: time.Now().UnixMilli(),
	}, true
}

func parseTags(raw string) map[string]string {
	tags := map[string]string{}
	for _, pair := range strings.Split(raw, ";") {
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) == 2 {
			tags[kv[0]] = kv[1]
		}
	}
	return tags
}

// parseEmotes parses the Twitch `emotes` tag format:
// emoteID:start-end,start-end/emoteID:start-end
func parseEmotes(raw, text string) []chatmsg.Emote {
	emotes := []chatmsg.Emote{}
	if raw == "" {
		return emotes
	}

	runes := []rune(text)

	for _, group := range strings.Split(raw, "/") {
		parts := strings.SplitN(group, ":", 2)
		if len(parts) != 2 {
			continue
		}
		id := parts[0]
		ranges := strings.Split(parts[1], ",")
		if len(ranges) == 0 {
			continue
		}
		first := strings.SplitN(ranges[0], "-", 2)
		if len(first) != 2 {
			continue
		}
		start, err1 := strconv.Atoi(first[0])
		end, err2 := strconv.Atoi(first[1])
		if err1 != nil || err2 != nil || start < 0 || end >= len(runes) || start > end {
			continue
		}
		code := string(runes[start : end+1])
		emotes = append(emotes, chatmsg.Emote{
			Code: code,
			URL:  fmt.Sprintf("https://static-cdn.jtvnw.net/emoticons/v2/%s/default/dark/2.0", id),
		})
	}

	return emotes
}
