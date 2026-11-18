package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFetchReleaseTagAuthenticates(t *testing.T) {
	t.Setenv("HOMEBREW_GITHUB_API_TOKEN", "test-token")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		fmt.Fprint(w, `{"tag_name":"v1.2.3"}`)
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if tag, err := fetchReleaseTag(ctx, server.URL); err != nil || tag != "v1.2.3" {
		t.Fatalf("tag = %q, error = %v", tag, err)
	}
}

func TestFetchReleaseTagRetriesWithoutRejectedToken(t *testing.T) {
	t.Setenv("HOMEBREW_GITHUB_API_TOKEN", "expired-token")
	for _, status := range []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Authorization") != "" {
					w.WriteHeader(status)
					return
				}
				fmt.Fprint(w, `{"tag_name":"v1.2.3"}`)
			}))
			defer server.Close()

			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			if tag, err := fetchReleaseTag(ctx, server.URL); err != nil || tag != "v1.2.3" {
				t.Fatalf("tag = %q, error = %v", tag, err)
			}
		})
	}
}

func TestVersionCacheExpiresFailuresSooner(t *testing.T) {
	now := time.Now()
	failed := versionCache{CheckedAt: now.Add(-time.Minute)}
	if !failed.fresh(now) || failed.fresh(now.Add(5*time.Minute)) {
		t.Fatal("a failed check should be cached for five minutes")
	}
	resolved := versionCache{Version: "v1.2.3", CheckedAt: failed.CheckedAt}
	if !resolved.fresh(now.Add(5*time.Minute)) || resolved.fresh(now.Add(24*time.Hour)) {
		t.Fatal("a successful check should be cached for one day")
	}
}

func TestFetchReleaseTagBoundsCredentialLookup(t *testing.T) {
	t.Setenv("HOMEBREW_GITHUB_API_TOKEN", "")
	bin := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "gh"), []byte("#!/bin/sh\nexec /bin/sleep 10\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Error("HTTP request started after credential lookup exhausted the deadline")
		fmt.Fprint(w, `{"tag_name":"v1.2.3"}`)
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if _, err := fetchReleaseTag(ctx, server.URL); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v, want context deadline exceeded", err)
	}
}
