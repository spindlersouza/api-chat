// Package config gerencia a configuracao local do app (config.json na pasta
// de config do usuario) e as credenciais de login da tela inicial.
package config

import (
	"bufio"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"api-chat/internal/secure"
)

// AppDirName nomeia a pasta de config local (%AppData%/<AppDirName>).
// Sobrescrita por -ldflags junto com DefaultAuthUser/DefaultAuthPass pra
// cada build sob encomenda, pra nao compartilhar config.json entre builds de
// clientes diferentes testadas na mesma conta do Windows, ex:
//   go build -ldflags "-X api-chat/internal/config.AppDirName=api-chat-fulano -X api-chat/internal/config.DefaultAuthUser=fulano -X api-chat/internal/config.DefaultAuthPass=segredo"
var (
	AppDirName      = "api-chat"
	DefaultAuthUser = "admin"
	DefaultAuthPass = "admin"
)

type Config struct {
	AuthUser string `json:"authUser"`
	AuthSalt string `json:"authSalt"`
	AuthHash string `json:"authHash"`

	Port              string   `json:"port"`
	TwitchChannel     string   `json:"twitchChannel"`
	YouTubeAPIKeyEnc  string   `json:"youtubeApiKeyEnc"`
	YouTubeChannelID  string   `json:"youtubeChannelId"`
	KickChannel       string   `json:"kickChannel"`
	KickChatroomID    string   `json:"kickChatroomId"`
	BotUsernames      []string `json:"botUsernames"`
	PollMinIntervalMs int      `json:"pollMinIntervalMs"`
	AlertCooldownMs   int      `json:"alertCooldownMs"`

	// YouTubeAPIKey e a versao decriptada, nunca persistida diretamente.
	YouTubeAPIKey string `json:"-"`
}

// Default retorna uma configuracao nova com usuario/senha admin/admin.
func Default() *Config {
	cfg := &Config{
		Port:              "3000",
		PollMinIntervalMs: 3000,
		AlertCooldownMs:   30000,
		BotUsernames:      []string{},
	}
	cfg.SetPassword(DefaultAuthUser, DefaultAuthPass)
	return cfg
}

func UserDataDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, AppDirName), nil
}

func Path() (string, error) {
	dir, err := UserDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

// Exists reporta se ja existe uma configuracao salva localmente.
func Exists() bool {
	p, err := Path()
	if err != nil {
		return false
	}
	_, err = os.Stat(p)
	return err == nil
}

// Load le a configuracao salva. Se nao existir ainda, cria uma configuracao
// default e tenta importar valores de um .env na pasta atual (compatibilidade
// com o setup anterior), sem persistir nada ainda.
func Load() *Config {
	if Exists() {
		if cfg, err := loadFromDisk(); err == nil {
			return cfg
		}
	}

	cfg := Default()
	importFromDotEnv(cfg)
	return cfg
}

func loadFromDisk() (*Config, error) {
	p, err := Path()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	if cfg.BotUsernames == nil {
		cfg.BotUsernames = []string{}
	}
	if key, err := secure.DecryptString(cfg.YouTubeAPIKeyEnc); err == nil {
		cfg.YouTubeAPIKey = key
	}
	return &cfg, nil
}

// Save persiste a configuracao localmente (a API key do YouTube e criptografada).
func (c *Config) Save() error {
	dir, err := UserDataDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	enc, err := secure.EncryptString(c.YouTubeAPIKey)
	if err != nil {
		enc = c.YouTubeAPIKey
	}
	out := *c
	out.YouTubeAPIKeyEnc = enc

	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}

	p, err := Path()
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o600)
}

// SetPassword define usuario/senha, guardando so um hash salgado (nunca a senha).
func (c *Config) SetPassword(user, password string) {
	salt := make([]byte, 16)
	_, _ = rand.Read(salt)
	c.AuthUser = user
	c.AuthSalt = base64.StdEncoding.EncodeToString(salt)
	c.AuthHash = hashPassword(salt, password)
}

// CheckPassword valida usuario/senha contra o hash salvo.
func (c *Config) CheckPassword(user, password string) bool {
	if subtle.ConstantTimeCompare([]byte(user), []byte(c.AuthUser)) != 1 {
		return false
	}
	salt, err := base64.StdEncoding.DecodeString(c.AuthSalt)
	if err != nil {
		return false
	}
	want := hashPassword(salt, password)
	return subtle.ConstantTimeCompare([]byte(want), []byte(c.AuthHash)) == 1
}

func hashPassword(salt []byte, password string) string {
	h := sha256.Sum256(append(append([]byte{}, salt...), []byte(password)...))
	return base64.StdEncoding.EncodeToString(h[:])
}

// BotUsernamesMap converte a lista salva pro formato usado pelo hub.
func (c *Config) BotUsernamesMap() map[string]bool {
	m := map[string]bool{}
	for _, name := range c.BotUsernames {
		name = strings.ToLower(strings.TrimSpace(name))
		if name != "" {
			m[name] = true
		}
	}
	return m
}

// BotUsernamesText junta a lista salva numa string separada por virgulas,
// pro campo de edicao na tela de config.
func (c *Config) BotUsernamesText() string {
	return strings.Join(c.BotUsernames, ", ")
}

// SetBotUsernamesText faz o caminho inverso de BotUsernamesText.
func (c *Config) SetBotUsernamesText(raw string) {
	names := []string{}
	for _, name := range strings.Split(raw, ",") {
		name = strings.TrimSpace(name)
		if name != "" {
			names = append(names, name)
		}
	}
	c.BotUsernames = names
}

// importFromDotEnv preenche cfg com valores de um .env na pasta atual, se
// existir (compatibilidade com configuracoes anteriores a tela de config).
func importFromDotEnv(cfg *Config) {
	env := map[string]string{}
	loadDotEnvInto(".env", env)
	if len(env) == 0 {
		return
	}

	if v := env["PORT"]; v != "" {
		cfg.Port = v
	}
	if v := env["TWITCH_CHANNEL"]; v != "" {
		cfg.TwitchChannel = strings.ToLower(strings.TrimSpace(v))
	}
	if v := env["YOUTUBE_API_KEY"]; v != "" {
		cfg.YouTubeAPIKey = v
	}
	if v := env["YOUTUBE_CHANNEL_ID"]; v != "" {
		cfg.YouTubeChannelID = v
	}
	if v := env["KICK_CHANNEL"]; v != "" {
		cfg.KickChannel = strings.ToLower(strings.TrimSpace(v))
	}
	if v := env["KICK_CHATROOM_ID"]; v != "" {
		cfg.KickChatroomID = v
	}
	if v := env["BOT_USERNAMES"]; v != "" {
		cfg.SetBotUsernamesText(v)
	}
	if v, err := strconv.Atoi(env["POLL_MIN_INTERVAL_MS"]); err == nil && v > 0 {
		cfg.PollMinIntervalMs = v
	}
	if v, err := strconv.Atoi(env["ALERT_COOLDOWN_MS"]); err == nil && v > 0 {
		cfg.AlertCooldownMs = v
	}
}

func loadDotEnvInto(path string, out map[string]string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.Trim(strings.TrimSpace(parts[1]), `"'`)
		out[key] = val
	}
}
