// Package slack posts messages to a Slack incoming webhook.
package slack

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

var httpClient = &http.Client{Timeout: 10 * time.Second}

// Message is the part of the incoming-webhook payload eo-cli uses. Text is the
// fallback shown in notifications and in clients that cannot render blocks, so
// it should carry the same information as Blocks.
type Message struct {
	Text   string  `json:"text"`
	Blocks []Block `json:"blocks,omitempty"`
}

type Block struct {
	Type     string `json:"type"`
	Text     *Text  `json:"text,omitempty"`
	Elements []Text `json:"elements,omitempty"`
}

type Text struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func markdown(s string) Text { return Text{Type: "mrkdwn", Text: s} }

// Section renders mrkdwn as a normal message paragraph.
func Section(mrkdwn string) Block {
	t := markdown(mrkdwn)
	return Block{Type: "section", Text: &t}
}

// Context renders mrkdwn smaller and greyed out, for secondary information.
func Context(mrkdwn string) Block {
	return Block{Type: "context", Elements: []Text{markdown(mrkdwn)}}
}

// Link renders an mrkdwn hyperlink.
func Link(url, label string) string {
	return fmt.Sprintf("<%s|%s>", url, Escape(label))
}

// Escape neutralises the three characters Slack reserves in mrkdwn, so that
// free text such as a justification cannot break the message layout.
func Escape(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;").Replace(s)
}

// Post sends msg to the given incoming-webhook URL. Slack answers a delivered
// message with "ok"; anything else is returned as an error, with the response
// body included because Slack puts its reason there (invalid_payload,
// no_service, channel_not_found) rather than in the status line.
func Post(webhookURL string, msg Message) error {
	if webhookURL == "" {
		return errors.New("empty webhook URL")
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, webhookURL, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := httpClient.Do(req)
	if err != nil {
		// Never return err itself: url.Error embeds the full request URL, and
		// that URL is the webhook secret.
		var timeoutErr interface{ Timeout() bool }
		if errors.As(err, &timeoutErr) && timeoutErr.Timeout() {
			return errors.New("Slack request timed out")
		}
		if cause := errors.Unwrap(err); cause != nil {
			return fmt.Errorf("Slack request failed: %w", cause)
		}
		return errors.New("Slack request failed")
	}
	defer res.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(res.Body, 1024))
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("Slack returned HTTP %d: %s", res.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}
