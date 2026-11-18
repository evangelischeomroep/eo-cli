package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	updateCheckURL     = "https://api.github.com/repos/evangelischeomroep/eo-cli/releases/latest"
	updateCheckTTL     = 24 * time.Hour
	updateCheckTimeout = 400 * time.Millisecond
	updateCheckWait    = 500 * time.Millisecond
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
	case <-time.After(updateCheckWait):
		return
	}
	if latest == "" || !isNewer(latest, current) {
		return
	}
	fmt.Fprintf(os.Stderr, "\n%s %s → %s\n", dim("update available:"), current, bold(latest))
	fmt.Fprintf(os.Stderr, "%s\n", dim("  install instructions: https://github.com/evangelischeomroep/eo-cli/releases/latest"))
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

	ctx, cancel := context.WithTimeout(context.Background(), updateCheckTimeout)
	defer cancel()
	latest, err := fetchReleaseTag(ctx, updateCheckURL)
	writeCache(versionCache{Version: latest, CheckedAt: time.Now()})
	return latest, err
}

func fetchReleaseTag(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "eo-cli/"+version)
	if token := githubReleaseToken(ctx); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
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

	return release.TagName, nil
}

func githubReleaseToken(ctx context.Context) string {
	if token := strings.TrimSpace(os.Getenv("HOMEBREW_GITHUB_API_TOKEN")); token != "" {
		return token
	}
	cmd := exec.CommandContext(ctx, "gh", "auth", "token", "--hostname", "github.com")
	cmd.WaitDelay = 25 * time.Millisecond
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
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
	err = json.Unmarshal(data, &c)
	return c, err
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
