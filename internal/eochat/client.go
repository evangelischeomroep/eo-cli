// Package eochat is a small client for EOchat, the Open WebUI instance hosted
// at https://chat.eo.nl. It covers the handful of endpoints the CLI needs:
// current user, model listing, knowledge bases, file uploads and
// (streaming) chat completions.
package eochat

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const DefaultBaseURL = "https://chat.eo.nl"

// ErrUnauthorized is returned (wrapped) when the server rejects the API key.
var ErrUnauthorized = errors.New("EOchat rejected the API key")

type Client struct {
	BaseURL   string
	APIKey    string
	UserAgent string
	HTTP      *http.Client
}

func NewClient(baseURL, apiKey string) *Client {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	// Chat completions can take a while to produce the first byte, but not forever.
	transport.ResponseHeaderTimeout = 2 * time.Minute
	return &Client{
		BaseURL:   strings.TrimRight(baseURL, "/"),
		APIKey:    apiKey,
		UserAgent: "eo-cli",
		HTTP:      &http.Client{Transport: transport},
	}
}

// APIError is a non-2xx response from EOchat.
type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	if msg := extractErrorMessage([]byte(e.Body)); msg != "" {
		return msg
	}
	if e.Body == "" {
		return fmt.Sprintf("HTTP %d", e.StatusCode)
	}
	return fmt.Sprintf("HTTP %d: %s", e.StatusCode, truncate(e.Body, 300))
}

// extractErrorMessage pulls a human-readable message out of the various error
// shapes Open WebUI produces: {"detail": "..."}, {"detail": [{"msg": ...}]},
// {"error": "..."} and {"error": {"message": "..."}}.
func extractErrorMessage(body []byte) string {
	var parsed struct {
		Detail json.RawMessage `json:"detail"`
		Error  json.RawMessage `json:"error"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return ""
	}
	if s := rawString(parsed.Detail); s != "" {
		return s
	}
	if s := rawString(parsed.Error); s != "" {
		return s
	}
	return ""
}

func rawString(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	var obj struct {
		Message string `json:"message"`
		Msg     string `json:"msg"`
	}
	if json.Unmarshal(raw, &obj) == nil {
		if obj.Message != "" {
			return obj.Message
		}
		if obj.Msg != "" {
			return obj.Msg
		}
	}
	var list []struct {
		Msg string `json:"msg"`
	}
	if json.Unmarshal(raw, &list) == nil && len(list) > 0 && list[0].Msg != "" {
		return list[0].Msg
	}
	return ""
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func (c *Client) newRequest(ctx context.Context, method, path string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.UserAgent)
	return req, nil
}

func (c *Client) send(req *http.Request) (*http.Response, error) {
	res, err := c.HTTP.Do(req)
	if err != nil {
		var timeoutErr interface{ Timeout() bool }
		if errors.As(err, &timeoutErr) && timeoutErr.Timeout() {
			return nil, fmt.Errorf("EOchat request timed out")
		}
		if errors.Is(err, context.Canceled) {
			return nil, err
		}
		if cause := errors.Unwrap(err); cause != nil {
			return nil, fmt.Errorf("EOchat request failed: %w", cause)
		}
		return nil, fmt.Errorf("EOchat request failed: %w", err)
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		defer res.Body.Close()
		errBody, _ := io.ReadAll(io.LimitReader(res.Body, 64<<10))
		apiErr := &APIError{StatusCode: res.StatusCode, Body: strings.TrimSpace(string(errBody))}
		if res.StatusCode == http.StatusUnauthorized || res.StatusCode == http.StatusForbidden {
			return nil, fmt.Errorf("%w: %v", ErrUnauthorized, apiErr)
		}
		return nil, apiErr
	}
	return res, nil
}

// do performs a JSON request and decodes the response into out (if non-nil).
func (c *Client) do(ctx context.Context, method, path string, body, out any) error {
	var reqBody io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reqBody = bytes.NewReader(buf)
	}
	req, err := c.newRequest(ctx, method, path, reqBody)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := c.send(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if out == nil {
		return nil
	}
	if err := json.NewDecoder(res.Body).Decode(out); err != nil {
		return fmt.Errorf("decoding EOchat response: %w", err)
	}
	return nil
}

// User is the account behind the API key.
type User struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
	Role  string `json:"role"`
}

// Me returns the user that owns the API key. Handy to validate a key.
func (c *Client) Me(ctx context.Context) (User, error) {
	var u User
	err := c.do(ctx, http.MethodGet, "/api/v1/auths/", nil, &u)
	return u, err
}

// Model is a model as listed by Open WebUI: base models from providers and
// custom "workspace" models built on top of them.
type Model struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	OwnedBy string `json:"owned_by"`
	Info    *struct {
		BaseModelID string `json:"base_model_id"`
		Meta        struct {
			Description string `json:"description"`
		} `json:"meta"`
	} `json:"info"`
	Tags []struct {
		Name string `json:"name"`
	} `json:"tags"`
}

func (m Model) Description() string {
	if m.Info == nil {
		return ""
	}
	return strings.TrimSpace(m.Info.Meta.Description)
}

// IsCustom reports whether this is a workspace model (a preset on top of a base model).
func (m Model) IsCustom() bool {
	return m.Info != nil && m.Info.BaseModelID != ""
}

// Models lists the models the current user may use.
func (c *Client) Models(ctx context.Context) ([]Model, error) {
	var out struct {
		Data []Model `json:"data"`
	}
	if err := c.do(ctx, http.MethodGet, "/api/models", nil, &out); err != nil {
		return nil, err
	}
	return out.Data, nil
}

// DefaultModel returns the server-side default model ID, or "" if none is set.
func (c *Client) DefaultModel(ctx context.Context) (string, error) {
	var out struct {
		DefaultModels string `json:"default_models"`
	}
	if err := c.do(ctx, http.MethodGet, "/api/config", nil, &out); err != nil {
		return "", err
	}
	for _, id := range strings.Split(out.DefaultModels, ",") {
		if id = strings.TrimSpace(id); id != "" {
			return id, nil
		}
	}
	return "", nil
}

// Knowledge is a knowledge base (document collection) usable for RAG.
type Knowledge struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// Knowledge lists the knowledge bases the current user can read.
func (c *Client) Knowledge(ctx context.Context) ([]Knowledge, error) {
	var raw json.RawMessage
	if err := c.do(ctx, http.MethodGet, "/api/v1/knowledge/", nil, &raw); err != nil {
		return nil, err
	}
	trimmed := bytes.TrimSpace(raw)
	// Older Open WebUI versions return a bare array, newer ones {items, total}.
	if len(trimmed) > 0 && trimmed[0] == '[' {
		var list []Knowledge
		if err := json.Unmarshal(trimmed, &list); err != nil {
			return nil, fmt.Errorf("decoding knowledge list: %w", err)
		}
		return list, nil
	}
	var out struct {
		Items []Knowledge `json:"items"`
	}
	if err := json.Unmarshal(trimmed, &out); err != nil {
		return nil, fmt.Errorf("decoding knowledge list: %w", err)
	}
	return out.Items, nil
}

// File is an uploaded file that can be attached to a chat for RAG.
type File struct {
	ID       string `json:"id"`
	Filename string `json:"filename"`
}

// UploadFile uploads a local file so it can be attached to a chat.
func (c *Client) UploadFile(ctx context.Context, path string) (File, error) {
	f, err := os.Open(path)
	if err != nil {
		return File{}, err
	}
	defer f.Close()

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	part, err := mw.CreateFormFile("file", filepath.Base(path))
	if err != nil {
		return File{}, err
	}
	if _, err := io.Copy(part, f); err != nil {
		return File{}, err
	}
	if err := mw.Close(); err != nil {
		return File{}, err
	}

	req, err := c.newRequest(ctx, http.MethodPost, "/api/v1/files/", &body)
	if err != nil {
		return File{}, err
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	res, err := c.send(req)
	if err != nil {
		return File{}, err
	}
	defer res.Body.Close()

	var uploaded File
	if err := json.NewDecoder(res.Body).Decode(&uploaded); err != nil {
		return File{}, fmt.Errorf("decoding upload response: %w", err)
	}
	return uploaded, nil
}
