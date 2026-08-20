package pim

import (
	"errors"
	"strings"
	"testing"
)

func TestSlackLabel(t *testing.T) {
	tests := []struct {
		name string
		in   Principal
		want string
	}{
		{"both", Principal{DisplayName: "Bob Mulder", Email: "bob.mulder@eo.nl"}, "*Bob Mulder* (bob.mulder@eo.nl)"},
		{"name only", Principal{DisplayName: "Bob Mulder"}, "*Bob Mulder*"},
		{"email only", Principal{Email: "bob.mulder@eo.nl"}, "*bob.mulder@eo.nl*"},
		{"neither", Principal{}, "An unidentified user"},
		// The raw object ID would mean nothing to a reader in Slack.
		{"id only", Principal{ID: "8a7b-1234"}, "An unidentified user"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := slackLabel(tt.in); got != tt.want {
				t.Errorf("slackLabel() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestActivationRequestedMessageNamesRequesterAndBothApprovalRoutes(t *testing.T) {
	msg := ActivationRequestedMessage(
		Principal{DisplayName: "Bob Mulder", Email: "bob.mulder@eo.nl"},
		"deploying release 2.4",
	)

	if !strings.Contains(msg.Text, "*Bob Mulder* (bob.mulder@eo.nl)") {
		t.Errorf("fallback text does not name the requester: %q", msg.Text)
	}

	all := msg.Text
	for _, b := range msg.Blocks {
		if b.Text != nil {
			all += "\n" + b.Text.Text
		}
		for _, e := range b.Elements {
			all += "\n" + e.Text
		}
	}

	for _, want := range []string{
		"*Bob Mulder* (bob.mulder@eo.nl)",
		"Contributor",
		"EO Studio Digitaal",
		ActivationDurationLabel,
		"deploying release 2.4",
		"eo pim approve",
		ApprovalPortalURL,
	} {
		if !strings.Contains(all, want) {
			t.Errorf("message is missing %q\n--- message ---\n%s", want, all)
		}
	}
}

func TestActivationRequestedMessageOmitsEmptyReason(t *testing.T) {
	msg := ActivationRequestedMessage(Principal{DisplayName: "Bob Mulder"}, "")
	for _, b := range msg.Blocks {
		if b.Text != nil && strings.Contains(b.Text.Text, "*Reason*") {
			t.Errorf("message has a Reason block for an empty justification: %q", b.Text.Text)
		}
	}
}

func TestActivationRequestedMessageEscapesJustification(t *testing.T) {
	msg := ActivationRequestedMessage(Principal{DisplayName: "Bob Mulder"}, "fix <b> & <i>")

	var reason string
	for _, b := range msg.Blocks {
		if b.Text != nil && strings.Contains(b.Text.Text, "*Reason*") {
			reason = b.Text.Text
		}
	}
	if reason == "" {
		t.Fatal("no Reason block found")
	}
	if strings.Contains(reason, "<b>") {
		t.Errorf("justification is not escaped for Slack mrkdwn: %q", reason)
	}
	if !strings.Contains(reason, "&lt;b&gt; &amp; &lt;i&gt;") {
		t.Errorf("justification escaped incorrectly: %q", reason)
	}
}

func TestWebhookLookupError(t *testing.T) {
	tests := []struct {
		name string
		az   string
		want string
	}{
		{
			// What az actually prints when the vault does not exist.
			name: "missing vault",
			az: "az keyvault secret show --vault-name kv-prod-eo-cli: ERROR: HTTPSConnectionPool(" +
				"host='kv-prod-eo-cli.vault.azure.net', port=443): Max retries exceeded (Caused by " +
				"NameResolutionError(\"<urllib3.connection.HTTPSConnection object at 0x103eda030>: " +
				"Failed to resolve 'kv-prod-eo-cli.vault.azure.net' ([Errno 8] nodename nor servname " +
				"provided, or not known)\"))",
			want: `Key Vault "kv-prod-eo-cli" does not exist, or is unreachable from this network`,
		},
		{
			name: "missing secret",
			az:   "ERROR: (SecretNotFound) A secret with (name/id) slack-pim-webhook was not found in this key vault.",
			want: `vault "kv-prod-eo-cli" has no secret "slack-pim-webhook" yet`,
		},
		{
			name: "no access",
			az:   "ERROR: (Forbidden) Caller is not authorized to perform action on resource.",
			want: `no read access to secret "slack-pim-webhook" in vault "kv-prod-eo-cli"`,
		},
		{
			name: "anything else keeps only the first line",
			az:   "ERROR: something unexpected\n  File \"/opt/az/lib/foo.py\", line 12\n    raise\n",
			want: `reading secret "slack-pim-webhook" from vault "kv-prod-eo-cli": ERROR: something unexpected`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := webhookLookupError(errors.New(tt.az)).Error()
			if got != tt.want {
				t.Errorf("webhookLookupError()\n got: %s\nwant: %s", got, tt.want)
			}
		})
	}
}

// The message a user sees must not carry az's multi-line plumbing.
func TestWebhookLookupErrorIsSingleLine(t *testing.T) {
	multiline := "ERROR: boom\nTraceback (most recent call last):\n  File \"x.py\"\n"
	if got := webhookLookupError(errors.New(multiline)).Error(); strings.Contains(got, "\n") {
		t.Errorf("error spans multiple lines: %q", got)
	}
}
