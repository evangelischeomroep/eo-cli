package slack

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestEscape(t *testing.T) {
	tests := []struct{ in, want string }{
		{"plain text", "plain text"},
		{"a & b", "a &amp; b"},
		{"<script>", "&lt;script&gt;"},
		{"fix <a|b> & go", "fix &lt;a|b&gt; &amp; go"},
	}
	for _, tt := range tests {
		if got := Escape(tt.in); got != tt.want {
			t.Errorf("Escape(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestLinkEscapesLabel(t *testing.T) {
	got := Link("https://example.com", "a & b")
	want := "<https://example.com|a &amp; b>"
	if got != want {
		t.Errorf("Link() = %q, want %q", got, want)
	}
}

func TestPostSendsJSONPayload(t *testing.T) {
	var gotBody []byte
	var gotContentType string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotBody, _ = io.ReadAll(r.Body)
		gotContentType = r.Header.Get("Content-Type")
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		w.Write([]byte("ok"))
	}))
	defer srv.Close()

	msg := Message{Text: "fallback", Blocks: []Block{Section("*hi*"), Context("ctx")}}
	if err := Post(srv.URL, msg); err != nil {
		t.Fatalf("Post() error = %v", err)
	}

	if gotContentType != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", gotContentType)
	}

	var decoded Message
	if err := json.Unmarshal(gotBody, &decoded); err != nil {
		t.Fatalf("payload is not valid JSON: %v", err)
	}
	if decoded.Text != "fallback" {
		t.Errorf("text = %q, want %q", decoded.Text, "fallback")
	}
	if len(decoded.Blocks) != 2 {
		t.Fatalf("got %d blocks, want 2", len(decoded.Blocks))
	}
	if decoded.Blocks[0].Type != "section" || decoded.Blocks[0].Text.Type != "mrkdwn" {
		t.Errorf("first block = %+v, want a mrkdwn section", decoded.Blocks[0])
	}
	if decoded.Blocks[1].Type != "context" || len(decoded.Blocks[1].Elements) != 1 {
		t.Errorf("second block = %+v, want a context block with one element", decoded.Blocks[1])
	}
}

func TestPostReturnsSlackError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("invalid_payload"))
	}))
	defer srv.Close()

	err := Post(srv.URL, Message{Text: "hi"})
	if err == nil {
		t.Fatal("Post() error = nil, want an error")
	}
	if !strings.Contains(err.Error(), "invalid_payload") {
		t.Errorf("error = %q, want it to mention invalid_payload", err)
	}
}

func TestPostRejectsEmptyURL(t *testing.T) {
	if err := Post("", Message{Text: "hi"}); err == nil {
		t.Fatal("Post(\"\") error = nil, want an error")
	}
}

// The webhook URL is a secret, so it must not surface in an error a caller
// prints to the terminal.
func TestPostErrorOmitsWebhookURL(t *testing.T) {
	const secret = "https://127.0.0.1:1/services/T000/B000/superSecretToken"

	err := Post(secret, Message{Text: "hi"})
	if err == nil {
		t.Fatal("Post() to a dead endpoint error = nil, want an error")
	}
	if strings.Contains(err.Error(), "superSecretToken") {
		t.Errorf("error leaks the webhook URL: %q", err)
	}
}
