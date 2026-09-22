package eochat

import (
	"os"
	"testing"
)

func TestConfigRoundTripAndEnv(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("EOCHAT_API_KEY", "")
	t.Setenv("EOCHAT_MODEL", "")
	t.Setenv("EOCHAT_URL", "")

	cfg, err := LoadConfig()
	if err != nil || cfg != (Config{}) {
		t.Fatalf("empty config expected: %+v %v", cfg, err)
	}
	if err := SaveConfig(Config{APIKey: "sk-1", Model: "m1"}); err != nil {
		t.Fatal(err)
	}
	path, _ := ConfigPath()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("config perms = %o, want 600", info.Mode().Perm())
	}
	cfg, err = LoadConfig()
	if err != nil || cfg.APIKey != "sk-1" || cfg.Model != "m1" {
		t.Fatalf("round trip: %+v %v", cfg, err)
	}
	if cfg.BaseURL() != DefaultBaseURL {
		t.Errorf("BaseURL = %q", cfg.BaseURL())
	}

	t.Setenv("EOCHAT_MODEL", "env-model")
	t.Setenv("EOCHAT_URL", "https://example.test/")
	eff := cfg.WithEnv()
	if eff.Model != "env-model" || eff.URL != "https://example.test/" || eff.APIKey != "sk-1" {
		t.Errorf("WithEnv = %+v", eff)
	}
}

func TestSessionRoundTrip(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if _, ok, err := LoadSession(); ok || err != nil {
		t.Fatalf("expected no session: %v %v", ok, err)
	}
	s := Session{Model: "m", Messages: []Message{{Role: "user", Content: "q"}, {Role: "assistant", Content: "a"}}}
	if err := SaveSession(s); err != nil {
		t.Fatal(err)
	}
	loaded, ok, err := LoadSession()
	if err != nil || !ok || len(loaded.Messages) != 2 || loaded.Model != "m" || loaded.UpdatedAt.IsZero() {
		t.Fatalf("LoadSession: %+v %v %v", loaded, ok, err)
	}
}
