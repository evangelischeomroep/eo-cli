package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	updateCheckURL = "https://api.github.com/repos/evangelischeomroep/eo-cli/releases/latest"
	updateCheckTTL = 24 * time.Hour
)

type versionCache struct {
	Version   string    `json:"version"`
	CheckedAt time.Time `json:"checked_at"`
}

func startUpdateCheck() <-chan string {
	ch := make(chan string, 1)
	go func() {
		latest, _ := fetchLatestVersion()
		ch <- latest
	}()
	return ch
}

func printUpdateNotice(ch <-chan string, current string) {
	if current == "dev" {
		return
	}
	var latest string
	select {
	case latest = <-ch:
	case <-time.After(500 * time.Millisecond):
		return
	}
	if latest == "" || normalize(latest) == normalize(current) {
		return
	}
	fmt.Fprintf(os.Stderr, "\n%s %s → %s\n", dim("update available:"), current, bold(latest))
	fmt.Fprintf(os.Stderr, "%s\n", dim("  curl -L https://github.com/evangelischeomroep/eo-cli/releases/latest/download/eo_darwin_arm64.tar.gz | tar xz && sudo mv eo /usr/local/bin/"))
}

func normalize(v string) string {
	return strings.TrimPrefix(v, "v")
}

func fetchLatestVersion() (string, error) {
	if cached, err := readCache(); err == nil && time.Since(cached.CheckedAt) < updateCheckTTL {
		return cached.Version, nil
	}

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(updateCheckURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}

	writeCache(versionCache{Version: release.TagName, CheckedAt: time.Now()})
	return release.TagName, nil
}

func cachePath() (string, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "eo", "update-check.json"), nil
}

func readCache() (versionCache, error) {
	path, err := cachePath()
	if err != nil {
		return versionCache{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return versionCache{}, err
	}
	var c versionCache
	return c, json.Unmarshal(data, &c)
}

func writeCache(c versionCache) {
	path, err := cachePath()
	if err != nil {
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}
	data, _ := json.Marshal(c)
	os.WriteFile(path, data, 0o644)
}
