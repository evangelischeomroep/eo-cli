package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
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
	if version == "dev" {
		ch <- ""
		return ch
	}
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
	if latest == "" || !isNewer(latest, current) {
		return
	}
	fmt.Fprintf(os.Stderr, "\n%s %s → %s\n", dim("update available:"), current, bold(latest))
	fmt.Fprintf(os.Stderr, "%s\n", dim("  download: https://github.com/evangelischeomroep/eo-cli/releases/latest"))
}

func isNewer(latest, current string) bool {
	l := parseSemver(normalize(latest))
	c := parseSemver(normalize(current))
	for i := range l {
		if l[i] != c[i] {
			return l[i] > c[i]
		}
	}
	return false
}

func parseSemver(v string) [3]int {
	parts := strings.SplitN(v, ".", 3)
	var out [3]int
	for i, p := range parts {
		if i >= 3 {
			break
		}
		out[i], _ = strconv.Atoi(p)
	}
	return out
}

func normalize(v string) string {
	return strings.TrimPrefix(v, "v")
}

func fetchLatestVersion() (string, error) {
	if cached, err := readCache(); err == nil && time.Since(cached.CheckedAt) < updateCheckTTL {
		return cached.Version, nil
	}

	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest(http.MethodGet, updateCheckURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "eo-cli/"+version)

	resp, err := client.Do(req)
	if err != nil {
		writeCache(versionCache{CheckedAt: time.Now()})
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		writeCache(versionCache{CheckedAt: time.Now()})
		return "", fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}
	if release.TagName == "" {
		return "", fmt.Errorf("empty tag_name in response")
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
