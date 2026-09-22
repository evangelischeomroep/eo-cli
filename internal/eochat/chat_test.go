package eochat

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestReadSSE(t *testing.T) {
	body := strings.Join([]string{
		": keep-alive",
		`data: {"choices":[{"delta":{"role":"assistant","content":""}}]}`,
		`data: {"choices":[{"delta":{"content":"Hallo"}}]}`,
		"",
		`data: {"choices":[{"delta":{"content":" wereld"}}]}`,
		`data: {"sources":[{"source":{"name":"handboek.pdf"}},{"metadata":[{"name":"adr-001.md"}]}]}`,
		`data: {"event":{"type":"status","data":{}}}`,
		"data: [DONE]",
		`data: {"choices":[{"delta":{"content":"ignored after done"}}]}`,
	}, "\n")

	var deltas []string
	result, err := readSSE(strings.NewReader(body), func(s string) { deltas = append(deltas, s) })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Content != "Hallo wereld" {
		t.Errorf("content = %q, want %q", result.Content, "Hallo wereld")
	}
	if len(deltas) != 2 {
		t.Errorf("deltas = %v, want 2 entries", deltas)
	}
	if len(result.Sources) != 2 || result.Sources[0].Name != "handboek.pdf" || result.Sources[1].Name != "adr-001.md" {
		t.Errorf("sources = %+v", result.Sources)
	}
}

func TestReadSSEError(t *testing.T) {
	body := "data: {\"choices\":[{\"delta\":{\"content\":\"partial\"}}]}\n" +
		"data: {\"error\":{\"message\":\"model overloaded\"}}\n"
	result, err := readSSE(strings.NewReader(body), nil)
	if err == nil || err.Error() != "model overloaded" {
		t.Fatalf("err = %v, want model overloaded", err)
	}
	if result.Content != "partial" {
		t.Errorf("partial content lost: %q", result.Content)
	}
}

func TestExtractErrorMessage(t *testing.T) {
	cases := map[string]string{
		`{"detail":"Model not found"}`:                  "Model not found",
		`{"detail":[{"msg":"field required"}]}`:         "field required",
		`{"error":"Unauthorized"}`:                      "Unauthorized",
		`{"error":{"message":"rate limited","code":1}}`: "rate limited",
		`not json`: "",
	}
	for in, want := range cases {
		if got := extractErrorMessage([]byte(in)); got != want {
			t.Errorf("extractErrorMessage(%s) = %q, want %q", in, got, want)
		}
	}
}

func newTestServer(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return NewClient(srv.URL, "sk-test")
}

func TestChatStreamSendsPayloadAndParsesStream(t *testing.T) {
	var got map[string]any
	client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat/completions" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer sk-test" {
			t.Errorf("missing bearer token: %q", r.Header.Get("Authorization"))
		}
		raw, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(raw, &got); err != nil {
			t.Fatalf("body not json: %v", err)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"ok\"}}]}\n\ndata: [DONE]\n")
	})

	result, err := client.ChatStream(context.Background(), ChatRequest{
		Model:     "gpt-test",
		Messages:  []Message{{Role: "user", Content: "hi"}},
		Files:     []Attachment{{Type: "collection", ID: "kb1"}},
		WebSearch: true,
	}, nil)
	if err != nil {
		t.Fatalf("ChatStream: %v", err)
	}
	if result.Content != "ok" {
		t.Errorf("content = %q", result.Content)
	}
	if got["model"] != "gpt-test" || got["stream"] != true {
		t.Errorf("payload = %v", got)
	}
	files, _ := got["files"].([]any)
	if len(files) != 1 {
		t.Errorf("files = %v", got["files"])
	}
	features, _ := got["features"].(map[string]any)
	if features["web_search"] != true {
		t.Errorf("features = %v", got["features"])
	}
}

func TestChatStreamFallsBackToJSON(t *testing.T) {
	client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"choices":[{"message":{"content":"non-streamed"}}]}`)
	})
	var deltas []string
	result, err := client.ChatStream(context.Background(), ChatRequest{Model: "m"}, func(s string) { deltas = append(deltas, s) })
	if err != nil {
		t.Fatalf("ChatStream: %v", err)
	}
	if result.Content != "non-streamed" || len(deltas) != 1 {
		t.Errorf("content=%q deltas=%v", result.Content, deltas)
	}
}

func TestUnauthorized(t *testing.T) {
	client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		io.WriteString(w, `{"detail":"Not authenticated"}`)
	})
	_, err := client.Me(context.Background())
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("err = %v, want ErrUnauthorized", err)
	}
	if !strings.Contains(err.Error(), "Not authenticated") {
		t.Errorf("error should carry server detail: %v", err)
	}
}

func TestModelsAndKnowledgeShapes(t *testing.T) {
	client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/models":
			io.WriteString(w, `{"data":[{"id":"eo-gpt","name":"EO GPT","info":{"base_model_id":"gpt-4o","meta":{"description":"Huisstijl"}}},{"id":"gpt-4o","name":"gpt-4o"}]}`)
		case "/api/v1/knowledge/":
			io.WriteString(w, `{"items":[{"id":"k1","name":"Handboek"}],"total":1}`)
		case "/api/config":
			io.WriteString(w, `{"default_models":"eo-gpt,gpt-4o"}`)
		default:
			http.NotFound(w, r)
		}
	})
	ctx := context.Background()
	models, err := client.Models(ctx)
	if err != nil || len(models) != 2 {
		t.Fatalf("Models: %v %v", models, err)
	}
	if !models[0].IsCustom() || models[0].Description() != "Huisstijl" || models[1].IsCustom() {
		t.Errorf("model metadata wrong: %+v", models)
	}
	kb, err := client.Knowledge(ctx)
	if err != nil || len(kb) != 1 || kb[0].Name != "Handboek" {
		t.Fatalf("Knowledge: %v %v", kb, err)
	}
	def, err := client.DefaultModel(ctx)
	if err != nil || def != "eo-gpt" {
		t.Fatalf("DefaultModel: %q %v", def, err)
	}
}

func TestKnowledgeLegacyArray(t *testing.T) {
	client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `[{"id":"k1","name":"Oud"}]`)
	})
	kb, err := client.Knowledge(context.Background())
	if err != nil || len(kb) != 1 {
		t.Fatalf("Knowledge legacy: %v %v", kb, err)
	}
}

func TestUploadFile(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/notes.txt"
	if err := writeFile(path, "hello"); err != nil {
		t.Fatal(err)
	}
	client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/files/" || r.Method != http.MethodPost {
			t.Errorf("%s %s", r.Method, r.URL.Path)
		}
		f, header, err := r.FormFile("file")
		if err != nil {
			t.Fatalf("FormFile: %v", err)
		}
		defer f.Close()
		data, _ := io.ReadAll(f)
		if string(data) != "hello" || header.Filename != "notes.txt" {
			t.Errorf("got %q %q", data, header.Filename)
		}
		io.WriteString(w, `{"id":"f1","filename":"notes.txt"}`)
	})
	uploaded, err := client.UploadFile(context.Background(), path)
	if err != nil || uploaded.ID != "f1" {
		t.Fatalf("UploadFile: %+v %v", uploaded, err)
	}
}

func TestResolveModel(t *testing.T) {
	models := []Model{{ID: "gpt-4o", Name: "GPT-4o"}, {ID: "eo-assistent", Name: "EO Assistent"}, {ID: "gpt-4o-mini", Name: "GPT-4o mini"}}
	if m, err := ResolveModel(models, "gpt-4o"); err != nil || m.ID != "gpt-4o" {
		t.Errorf("exact id: %v %v", m, err)
	}
	if m, err := ResolveModel(models, "eo assistent"); err != nil || m.ID != "eo-assistent" {
		t.Errorf("name: %v %v", m, err)
	}
	if m, err := ResolveModel(models, "mini"); err != nil || m.ID != "gpt-4o-mini" {
		t.Errorf("substring: %v %v", m, err)
	}
	if _, err := ResolveModel(models, "gpt"); err == nil {
		t.Errorf("ambiguous match should fail")
	}
	if _, err := ResolveModel(models, "claude"); err == nil {
		t.Errorf("no match should fail")
	}
}

func TestSessionPromptMessages(t *testing.T) {
	s := Session{System: "be brief", Messages: []Message{{Role: "user", Content: "hi"}}}
	msgs := s.PromptMessages()
	if len(msgs) != 2 || msgs[0].Role != "system" || msgs[1].Content != "hi" {
		t.Errorf("PromptMessages = %+v", msgs)
	}
}
