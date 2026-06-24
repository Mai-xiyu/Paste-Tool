package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Mai-xiyu/Paste-Tool/internal/metadata"
)

const (
	DefaultStartDelayMS    = 3000
	DefaultInterKeyDelayMS = 8
	DefaultBatchSize       = 50
	DefaultBatchPauseMS    = 20
)

var allowedModifiers = map[string]string{
	"ctrl":    "Ctrl",
	"control": "Ctrl",
	"alt":     "Alt",
	"shift":   "Shift",
	"win":     "Win",
	"super":   "Win",
	"cmd":     "Win",
	"meta":    "Win",
}

type Config struct {
	Hotkey HotkeyConfig `json:"hotkey"`
	Paste  PasteConfig  `json:"paste"`
	Update UpdateConfig `json:"update"`
	UI     UIConfig     `json:"ui"`
}

type HotkeyConfig struct {
	Modifiers []string `json:"modifiers"`
	Key       string   `json:"key"`
}

type PasteConfig struct {
	StartDelayMS    int `json:"start_delay_ms"`
	InterKeyDelayMS int `json:"inter_key_delay_ms"`
	BatchSize       int `json:"batch_size"`
	BatchPauseMS    int `json:"batch_pause_ms"`
}

type UpdateConfig struct {
	Repository string `json:"repository"`
}

type UIConfig struct {
	Language string `json:"language"`
}

func Default() Config {
	return Config{
		Hotkey: HotkeyConfig{
			Modifiers: []string{"Ctrl", "Alt"},
			Key:       "V",
		},
		Paste: PasteConfig{
			StartDelayMS:    DefaultStartDelayMS,
			InterKeyDelayMS: DefaultInterKeyDelayMS,
			BatchSize:       DefaultBatchSize,
			BatchPauseMS:    DefaultBatchPauseMS,
		},
		Update: UpdateConfig{
			Repository: "Mai-xiyu/Paste-Tool",
		},
		UI: UIConfig{
			Language: "auto",
		},
	}
}

func ConfigPath() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("locate user config dir: %w", err)
	}
	return filepath.Join(base, "Mai-xiyu", "PasteTool", "config.json"), nil
}

func LoadDefault() (Config, string, error) {
	path, err := ConfigPath()
	if err != nil {
		return Default(), "", err
	}
	cfg, err := Load(path)
	return cfg, path, err
}

func Load(path string) (Config, error) {
	cfg := Default()
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return cfg, fmt.Errorf("read config %q: %w", path, err)
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return cfg, nil
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parse config %q: %w", path, err)
	}
	cfg.applyDefaults()
	if err := cfg.Validate(); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func Save(path string, cfg Config) error {
	cfg.applyDefaults()
	if err := cfg.Validate(); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(data, '\n'), 0o600); err != nil {
		return fmt.Errorf("write temp config: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("replace config: %w", err)
	}
	return nil
}

func (c *Config) applyDefaults() {
	d := Default()
	if len(c.Hotkey.Modifiers) == 0 {
		c.Hotkey.Modifiers = d.Hotkey.Modifiers
	}
	if strings.TrimSpace(c.Hotkey.Key) == "" {
		c.Hotkey.Key = d.Hotkey.Key
	}
	if strings.TrimSpace(c.Update.Repository) == "" {
		c.Update.Repository = d.Update.Repository
	}
	if strings.TrimSpace(c.UI.Language) == "" {
		c.UI.Language = d.UI.Language
	}
	c.normalize()
}

func (c *Config) normalize() {
	c.Hotkey.Key = normalizeKey(c.Hotkey.Key)
	c.Hotkey.Modifiers = normalizeModifiers(c.Hotkey.Modifiers)
	c.UI.Language = normalizeLanguage(c.UI.Language)
}

func (c Config) Validate() error {
	if len(c.Hotkey.Modifiers) == 0 {
		return errors.New("hotkey requires at least one modifier")
	}
	if normalizeKey(c.Hotkey.Key) == "" {
		return errors.New("hotkey key is required")
	}
	if c.Paste.StartDelayMS < 0 {
		return errors.New("paste.start_delay_ms must be >= 0")
	}
	if c.Paste.InterKeyDelayMS < 0 {
		return errors.New("paste.inter_key_delay_ms must be >= 0")
	}
	if c.Paste.BatchSize < 0 {
		return errors.New("paste.batch_size must be >= 0")
	}
	if c.Paste.BatchPauseMS < 0 {
		return errors.New("paste.batch_pause_ms must be >= 0")
	}
	if strings.Count(c.Update.Repository, "/") != 1 {
		return errors.New("update.repository must use owner/repo format")
	}
	if normalizeLanguage(c.UI.Language) == "" {
		return errors.New("ui.language must be auto, zh-CN, or en")
	}
	return nil
}

func (c Config) HotkeyString() string {
	parts := append([]string{}, c.Hotkey.Modifiers...)
	parts = append(parts, normalizeKey(c.Hotkey.Key))
	return strings.Join(parts, "+")
}

func (p PasteConfig) StartDelay() time.Duration {
	return time.Duration(p.StartDelayMS) * time.Millisecond
}

func (p PasteConfig) InterKeyDelay() time.Duration {
	return time.Duration(p.InterKeyDelayMS) * time.Millisecond
}

func (p PasteConfig) BatchPause() time.Duration {
	return time.Duration(p.BatchPauseMS) * time.Millisecond
}

func (c Config) Get(key string) (string, error) {
	switch normalizePathKey(key) {
	case "", "all":
		data, err := json.MarshalIndent(c, "", "  ")
		if err != nil {
			return "", err
		}
		return string(data), nil
	case "hotkey":
		return c.HotkeyString(), nil
	case "hotkey.modifiers":
		return strings.Join(c.Hotkey.Modifiers, "+"), nil
	case "hotkey.key":
		return c.Hotkey.Key, nil
	case "paste.start_delay_ms":
		return strconv.Itoa(c.Paste.StartDelayMS), nil
	case "paste.inter_key_delay_ms":
		return strconv.Itoa(c.Paste.InterKeyDelayMS), nil
	case "paste.batch_size":
		return strconv.Itoa(c.Paste.BatchSize), nil
	case "paste.batch_pause_ms":
		return strconv.Itoa(c.Paste.BatchPauseMS), nil
	case "update.repository":
		return c.Update.Repository, nil
	case "ui.language":
		return c.UI.Language, nil
	default:
		return "", fmt.Errorf("unknown config key %q", key)
	}
}

func (c *Config) Set(key, value string) error {
	value = strings.TrimSpace(value)
	switch normalizePathKey(key) {
	case "hotkey":
		parts := strings.Split(value, "+")
		if len(parts) < 2 {
			return errors.New("hotkey must use a form like Ctrl+Alt+V")
		}
		c.Hotkey.Modifiers = normalizeModifiers(parts[:len(parts)-1])
		c.Hotkey.Key = normalizeKey(parts[len(parts)-1])
	case "hotkey.modifiers":
		c.Hotkey.Modifiers = normalizeModifiers(strings.FieldsFunc(value, func(r rune) bool {
			return r == '+' || r == ',' || r == ' '
		}))
	case "hotkey.key":
		c.Hotkey.Key = normalizeKey(value)
	case "paste.start_delay_ms":
		n, err := parseNonNegativeInt(key, value)
		if err != nil {
			return err
		}
		c.Paste.StartDelayMS = n
	case "paste.inter_key_delay_ms":
		n, err := parseNonNegativeInt(key, value)
		if err != nil {
			return err
		}
		c.Paste.InterKeyDelayMS = n
	case "paste.batch_size":
		n, err := parseNonNegativeInt(key, value)
		if err != nil {
			return err
		}
		c.Paste.BatchSize = n
	case "paste.batch_pause_ms":
		n, err := parseNonNegativeInt(key, value)
		if err != nil {
			return err
		}
		c.Paste.BatchPauseMS = n
	case "update.repository":
		c.Update.Repository = value
	case "ui.language":
		lang := normalizeLanguage(value)
		if lang == "" {
			return errors.New("ui.language must be auto, zh-CN, or en")
		}
		c.UI.Language = lang
	default:
		return fmt.Errorf("unknown config key %q", key)
	}
	c.normalize()
	return c.Validate()
}

func normalizePathKey(key string) string {
	return strings.ToLower(strings.TrimSpace(strings.ReplaceAll(key, "-", "_")))
}

func parseNonNegativeInt(name, value string) (int, error) {
	n, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer: %w", name, err)
	}
	if n < 0 {
		return 0, fmt.Errorf("%s must be >= 0", name)
	}
	return n, nil
}

func normalizeModifiers(values []string) []string {
	seen := make(map[string]bool, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		canonical, ok := allowedModifiers[strings.ToLower(strings.TrimSpace(value))]
		if !ok || seen[canonical] {
			continue
		}
		seen[canonical] = true
		out = append(out, canonical)
	}
	return out
}

func normalizeKey(key string) string {
	key = strings.ToUpper(strings.TrimSpace(key))
	if len(key) == 1 {
		ch := key[0]
		if (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') {
			return key
		}
	}
	if strings.HasPrefix(key, "F") {
		n, err := strconv.Atoi(strings.TrimPrefix(key, "F"))
		if err == nil && n >= 1 && n <= 12 {
			return fmt.Sprintf("F%d", n)
		}
	}
	return ""
}

func normalizeLanguage(value string) string {
	value = strings.TrimSpace(strings.ReplaceAll(value, "_", "-"))
	if value == "" {
		return "auto"
	}
	switch strings.ToLower(value) {
	case "auto", "system":
		return "auto"
	case "zh", "zh-cn", "zh-hans", "zh-hans-cn", "cn":
		return "zh-CN"
	case "en", "en-us", "en-gb":
		return "en"
	default:
		return ""
	}
}

func DefaultConfigPathForDisplay() string {
	path, err := ConfigPath()
	if err != nil {
		return filepath.Join("<config-dir>", "Mai-xiyu", "PasteTool", "config.json")
	}
	return path
}

func ResetToDefaults(path string) error {
	if path == "" {
		var err error
		path, err = ConfigPath()
		if err != nil {
			return err
		}
	}
	return Save(path, Default())
}

func RepositoryURL(cfg Config) string {
	repo := strings.TrimSpace(cfg.Update.Repository)
	if repo == "" {
		return metadata.Repository
	}
	return "https://github.com/" + repo
}
