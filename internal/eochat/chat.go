package eochat

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Attachment references an uploaded file ("file") or a knowledge base
// ("collection") that Open WebUI should use to ground the answer.
type Attachment struct {
	Type string `json:"type"`
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
}

type ChatRequest struct {
	Model     string
	Messages  []Message
	Files     []Attachment
	WebSearch bool
}

// Source is a citation Open WebUI returned alongside the answer.
type Source struct {
	Name string
}

type ChatResult struct {
	Content string
	Sources []Source
}

type chatPayload struct {
	Model    string         `json:"model"`
	Messages []Message      `json:"messages"`
	Stream   bool           `json:"stream"`
	Files    []Attachment   `json:"files,omitempty"`
	Features map[string]any `json:"features,omitempty"`
}

func buildPayload(req ChatRequest, stream bool) chatPayload {
	p := chatPayload{Model: req.Model, Messages: req.Messages, Stream: stream, Files: req.Files}
	if req.WebSearch {
		p.Features = map[string]any{"web_search": true}
	}
	return p
}

// Chat performs a non-streaming completion.
func (c *Client) Chat(ctx context.Context, req ChatRequest) (ChatResult, error) {
	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Sources []rawSource `json:"sources"`
	}
	if err := c.do(ctx, http.MethodPost, "/api/chat/completions", buildPayload(req, false), &out); err != nil {
		return ChatResult{}, err
	}
	if len(out.Choices) == 0 {
		return ChatResult{}, errors.New("EOchat returned no answer")
	}
	return ChatResult{Content: out.Choices[0].Message.Content, Sources: convertSources(out.Sources)}, nil
}

// ChatStream performs a streaming completion, calling onDelta for every piece
// of text as it arrives. The full answer is returned when the stream ends.
// A cancelled context returns the partial answer together with ctx.Err().
func (c *Client) ChatStream(ctx context.Context, req ChatRequest, onDelta func(string)) (ChatResult, error) {
	buf, err := json.Marshal(buildPayload(req, true))
	if err != nil {
		return ChatResult{}, err
	}
	httpReq, err := c.newRequest(ctx, http.MethodPost, "/api/chat/completions", strings.NewReader(string(buf)))
	if err != nil {
		return ChatResult{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	res, err := c.send(httpReq)
	if err != nil {
		return ChatResult{}, err
	}
	defer res.Body.Close()

	if !strings.HasPrefix(res.Header.Get("Content-Type"), "text/event-stream") {
		// Some models/pipelines ignore stream=true and answer in one go.
		var out struct {
			Choices []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			} `json:"choices"`
			Sources []rawSource `json:"sources"`
		}
		if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
			return ChatResult{}, fmt.Errorf("decoding EOchat response: %w", err)
		}
		if len(out.Choices) == 0 {
			return ChatResult{}, errors.New("EOchat returned no answer")
		}
		if onDelta != nil {
			onDelta(out.Choices[0].Message.Content)
		}
		return ChatResult{Content: out.Choices[0].Message.Content, Sources: convertSources(out.Sources)}, nil
	}

	result, err := readSSE(res.Body, onDelta)
	if err != nil && ctx.Err() != nil {
		return result, ctx.Err()
	}
	return result, err
}

type rawSource struct {
	Source struct {
		Name string `json:"name"`
		ID   string `json:"id"`
	} `json:"source"`
	Metadata []struct {
		Name   string `json:"name"`
		Source string `json:"source"`
	} `json:"metadata"`
}

func convertSources(raw []rawSource) []Source {
	seen := map[string]struct{}{}
	var out []Source
	add := func(name string) {
		name = strings.TrimSpace(name)
		if name == "" {
			return
		}
		if _, ok := seen[name]; ok {
			return
		}
		seen[name] = struct{}{}
		out = append(out, Source{Name: name})
	}
	for _, s := range raw {
		if s.Source.Name != "" {
			add(s.Source.Name)
			continue
		}
		for _, m := range s.Metadata {
			if m.Name != "" {
				add(m.Name)
			} else {
				add(m.Source)
			}
		}
	}
	return out
}

type streamChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
	Sources []rawSource     `json:"sources"`
	Error   json.RawMessage `json:"error"`
	Detail  json.RawMessage `json:"detail"`
}

// readSSE consumes an OpenAI-style server-sent-events body. Lines that are not
// "data:" lines, and data objects we do not understand, are ignored.
func readSSE(body io.Reader, onDelta func(string)) (ChatResult, error) {
	var result ChatResult
	var content strings.Builder
	var sources []rawSource

	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 64<<10), 16<<20)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ":") || !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "[DONE]" {
			break
		}
		var chunk streamChunk
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			continue
		}
		if msg := rawString(chunk.Error); msg != "" {
			result.Content = content.String()
			return result, errors.New(msg)
		}
		if msg := rawString(chunk.Detail); msg != "" && len(chunk.Choices) == 0 {
			result.Content = content.String()
			return result, errors.New(msg)
		}
		if len(chunk.Sources) > 0 {
			sources = append(sources, chunk.Sources...)
		}
		for _, choice := range chunk.Choices {
			if choice.Delta.Content == "" {
				continue
			}
			content.WriteString(choice.Delta.Content)
			if onDelta != nil {
				onDelta(choice.Delta.Content)
			}
		}
	}
	result.Content = content.String()
	result.Sources = convertSources(sources)
	if err := scanner.Err(); err != nil {
		return result, err
	}
	return result, nil
}
