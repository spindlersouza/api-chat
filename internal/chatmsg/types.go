package chatmsg

type Emote struct {
	Code string `json:"code"`
	URL  string `json:"url"`
}

type Message struct {
	Platform  string  `json:"platform"` // "twitch", "youtube" or "kick"
	Username  string  `json:"username"`
	Message   string  `json:"message"`
	Avatar    string  `json:"avatar"`
	Badges    []string `json:"badges"`
	Color     string  `json:"color"`
	Emotes    []Emote `json:"emotes"`
	Timestamp int64   `json:"timestamp"`
}
