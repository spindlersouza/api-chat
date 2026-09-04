// Package config gerencia a configuracao local do app: um config.json
// criptografado (DPAPI do Windows) guardado em %AppData%, numa subpasta
// derivada do caminho do proprio executavel.
package config

import (
	"bufio"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"api-chat/internal/secure"
)

// DefaultAuthUser/DefaultAuthPass definem o login padrao de uma build.
// Sobrescritos por -ldflags em builds sob encomenda pra cada cliente, ex:
//   go build -ldflags "-X api-chat/internal/config.DefaultAuthUser=fulano -X api-chat/internal/config.DefaultAuthPass=segredo"
var (
	DefaultAuthUser = "admin"
	DefaultAuthPass = "admin"
)

type Config struct {
	AuthUser string `json:"authUser"`
	AuthSalt string `json:"authSalt"`
	AuthHash string `json:"authHash"`

	Port              string   `json:"port"`
	TwitchChannel     string   `json:"twitchChannel"`
	YouTubeAPIKey     string   `json:"youtubeApiKey"`
	YouTubeChannelID  string   `json:"youtubeChannelId"`
	KickChannel       string   `json:"kickChannel"`
	KickChatroomID    string   `json:"kickChatroomId"`
	BotUsernames      []string `json:"botUsernames"`
	PollMinIntervalMs int      `json:"pollMinIntervalMs"`
	AlertCooldownMs   int      `json:"alertCooldownMs"`
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

// UserDataDir retorna a pasta de config dessa instalacao, dentro de
// %AppData%\api-chat, numa subpasta derivada do caminho absoluto do proprio
// executavel — cada copia do .exe (dev, release, cliente X) fica isolada
// automaticamente, sem precisar de flag manual nenhuma, e o arquivo fica
// escondido no perfil do usuario em vez de do lado do .exe.
func UserDataDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}

	exePath, err := os.Executable()
	if err != nil {
		return "", err
	}
	exePath, err = filepath.Abs(exePath)
	if err != nil {
		return "", err
	}

	sum := sha256.Sum256([]byte(strings.ToLower(exePath)))
	id := hex.EncodeToString(sum[:])[:12]

	return filepath.Join(base, "api-chat", id), nil
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
	encrypted, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}

	data, err := secure.DecryptString(string(encrypted))
	if err != nil {
		return nil, fmt.Errorf("decrypting config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal([]byte(data), &cfg); err != nil {
		return nil, err
	}
	if cfg.BotUsernames == nil {
		cfg.BotUsernames = []string{}
	}
	return &cfg, nil
}

// Save persiste a configuracao localmente. O arquivo inteiro e criptografado
// com DPAPI do Windows (so decriptografa na mesma conta que salvou).
func (c *Config) Save() error {
	dir, err := UserDataDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	encrypted, err := secure.EncryptString(string(data))
	if err != nil {
		return fmt.Errorf("encrypting config: %w", err)
	}

	p, err := Path()
	if err != nil {
		return err
	}
	return os.WriteFile(p, []byte(encrypted), 0o600)
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

// importFromDotEnv preenche cfg com valores de um .env ao lado do executavel,
// se existir (compatibilidade com configuracoes anteriores a tela de
// config). Usa a pasta do .exe, nao o diretorio de trabalho do processo, pra
// nao herdar um .env de onde o app foi lancado.
func importFromDotEnv(cfg *Config) {
	envPath := ".env"
	if exePath, err := os.Executable(); err == nil {
		envPath = filepath.Join(filepath.Dir(exePath), ".env")
	}

	env := map[string]string{}
	loadDotEnvInto(envPath, env)
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
