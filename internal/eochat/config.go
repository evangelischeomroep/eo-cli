package eochat

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Config is what `eo ask login` and `eo ask model` persist. Environment
// variables (EOCHAT_URL, EOCHAT_API_KEY, EOCHAT_MODEL) override the file.
type Config struct {
	URL    string `json:"url,omitempty"`
	APIKey string `json:"api_key,omitempty"`
	Model  string `json:"model,omitempty"`
}

// ConfigDir is ~/.config/eo (or $XDG_CONFIG_HOME/eo).
func ConfigDir() (string, error) {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "eo"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "eo"), nil
}

func ConfigPath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "eochat.json"), nil
}

// LoadConfig reads the config file. A missing file yields an empty config.
func LoadConfig() (Config, error) {
	path, err := ConfigPath()
	if err != nil {
		return Config{}, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Config{}, nil
	}
	if err != nil {
		return Config{}, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("reading %s: %w", path, err)
	}
	return cfg, nil
}

func SaveConfig(cfg Config) error {
	path, err := ConfigPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o600)
}

// WithEnv returns a copy with environment overrides applied.
func (c Config) WithEnv() Config {
	if v := os.Getenv("EOCHAT_URL"); v != "" {
		c.URL = v
	}
	if v := os.Getenv("EOCHAT_API_KEY"); v != "" {
		c.APIKey = v
	}
	if v := os.Getenv("EOCHAT_MODEL"); v != "" {
		c.Model = v
	}
	return c
}

func (c Config) BaseURL() string {
	if c.URL == "" {
		return DefaultBaseURL
	}
	return c.URL
}

// Session is a local conversation so `eo ask -c` can pick up where you left off.
type Session struct {
	Model     string       `json:"model"`
	System    string       `json:"system,omitempty"`
	Messages  []Message    `json:"messages"`
	Files     []Attachment `json:"files,omitempty"`
	UpdatedAt time.Time    `json:"updated_at"`
}

func SessionPath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "eochat-session.json"), nil
}

// LoadSession returns the last saved conversation. ok is false if there is none.
func LoadSession() (s Session, ok bool, err error) {
	path, err := SessionPath()
	if err != nil {
		return Session{}, false, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Session{}, false, nil
	}
	if err != nil {
		return Session{}, false, err
	}
	if err := json.Unmarshal(data, &s); err != nil {
		return Session{}, false, fmt.Errorf("reading %s: %w", path, err)
	}
	return s, true, nil
}

func SaveSession(s Session) error {
	path, err := SessionPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	s.UpdatedAt = time.Now()
	data, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// PromptMessages returns the messages to send: the system prompt (if any)
// followed by the conversation.
func (s Session) PromptMessages() []Message {
	msgs := make([]Message, 0, len(s.Messages)+1)
	if s.System != "" {
		msgs = append(msgs, Message{Role: "system", Content: s.System})
	}
	return append(msgs, s.Messages...)
}
