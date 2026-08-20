package pim

import (
	"fmt"
	"strings"

	"github.com/evangelischeomroep/eo-cli/internal/azure"
	"github.com/evangelischeomroep/eo-cli/internal/slack"
)

const (
	// WebhookVaultName and WebhookSecretName locate the Slack incoming-webhook
	// URL. The URL itself is a secret and cannot live here: eo-cli is a public
	// repository and its releases are public binaries, so a committed or
	// ldflags-injected URL would be readable by anyone.
	WebhookVaultName  = "kv-eonl-shared-001"
	WebhookSecretName = "slack-pim-webhook"

	// ApprovalPortalURL is the PIM "Approve requests" blade for Azure resource
	// roles, which is where an approver lands to act on the request.
	ApprovalPortalURL = "https://entra.microsoft.com/#view/Microsoft_Azure_PIMCommon/ApprovalMenuBlade/~/azurerbac"
)

// slackLabel renders a principal for a Slack message, bolding the display name
// but not the email so the eye lands on the person. Principal.String() is not
// reused because it falls back to the raw object ID, which means nothing to a
// reader in Slack.
func slackLabel(p Principal) string {
	name, email := slack.Escape(p.DisplayName), slack.Escape(p.Email)
	switch {
	case name != "" && email != "":
		return fmt.Sprintf("*%s* (%s)", name, email)
	case name != "":
		return fmt.Sprintf("*%s*", name)
	case email != "":
		return fmt.Sprintf("*%s*", email)
	default:
		return "An unidentified user"
	}
}

// SlackWebhookURL fetches the incoming-webhook URL from Key Vault.
func SlackWebhookURL() (string, error) {
	url, err := azure.GetKeyVaultSecret(WebhookVaultName, WebhookSecretName)
	if err != nil {
		return "", webhookLookupError(err)
	}
	if url == "" {
		return "", fmt.Errorf("secret %q in vault %q is empty", WebhookSecretName, WebhookVaultName)
	}
	return url, nil
}

// webhookLookupError names what actually went wrong. The az CLI reports the
// three cases that matter here as multi-line Python plumbing — a missing vault
// arrives as a urllib3 DNS stack — which says nothing to someone who just
// wanted to know why Slack stayed quiet.
func webhookLookupError(err error) error {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "Failed to resolve"), strings.Contains(msg, "NameResolutionError"):
		return fmt.Errorf("Key Vault %q does not exist, or is unreachable from this network", WebhookVaultName)
	case strings.Contains(msg, "SecretNotFound"):
		return fmt.Errorf("vault %q has no secret %q yet", WebhookVaultName, WebhookSecretName)
	case strings.Contains(msg, "Forbidden"), strings.Contains(msg, "AccessDenied"):
		return fmt.Errorf("no read access to secret %q in vault %q", WebhookSecretName, WebhookVaultName)
	}
	return fmt.Errorf("reading secret %q from vault %q: %s", WebhookSecretName, WebhookVaultName, firstLine(msg))
}

// firstLine keeps an az error to one terminal line; the rest is a stack trace.
func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return strings.TrimSpace(s[:i])
	}
	return strings.TrimSpace(s)
}

// ActivationRequestedMessage announces that someone asked for the Contributor
// role and points approvers at the two ways to act on it.
func ActivationRequestedMessage(requester Principal, justification string) slack.Message {
	headline := fmt.Sprintf(
		":lock: %s requests the *Contributor* role on *%s* for %s.",
		slackLabel(requester), slack.Escape(azure.SubscriptionName), ActivationDurationLabel,
	)

	blocks := []slack.Block{slack.Section(headline)}
	if justification != "" {
		blocks = append(blocks, slack.Section("*Reason*\n"+slack.Escape(justification)))
	}
	blocks = append(blocks, slack.Context(fmt.Sprintf(
		"Approve with `eo pim approve` or in %s",
		slack.Link(ApprovalPortalURL, "Privileged Identity Management"),
	)))

	return slack.Message{Text: headline, Blocks: blocks}
}

// NotifyActivationRequested posts the request announcement to Slack.
func NotifyActivationRequested(webhookURL string, requester Principal, justification string) error {
	return slack.Post(webhookURL, ActivationRequestedMessage(requester, justification))
}
