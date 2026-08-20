# eo-cli

Developer CLI for Evangelische Omroep. Currently focused on Azure PIM: activating your own role and approving requests from teammates.

## Installation

### Homebrew (macOS and Linux, recommended)

Install with Homebrew:

```bash
brew tap evangelischeomroep/eo-cli https://github.com/evangelischeomroep/eo-cli
brew install --cask evangelischeomroep/eo-cli/eo
```

Homebrew also installs the Azure CLI dependency and shell completions. Sign in before using `eo`:

```bash
az login
```

### macOS (manual)

Download the correct binary from the [Releases page](https://github.com/evangelischeomroep/eo-cli/releases/latest):

| Mac | File |
|-----|------|
| Apple Silicon (M1/M2/M3) | `eo_darwin_arm64.tar.gz` |
| Intel | `eo_darwin_amd64.tar.gz` |

```bash
# Replace the URL with the correct architecture for your machine
curl -L https://github.com/evangelischeomroep/eo-cli/releases/latest/download/eo_darwin_arm64.tar.gz | tar xz
sudo mv eo /usr/local/bin/
```

Verify the installation:

```bash
eo version
```

### Via Go (developers)

If you have Go installed:

```bash
go install github.com/evangelischeomroep/eo-cli/cmd/eo@latest
```

## Requirements

- [Azure CLI](https://learn.microsoft.com/en-us/cli/azure/install-azure-cli) installed and logged in (`az login`)
- Access to the _EO Studio Digitaal_ subscription
- For `pim approve`: you must be an approver on the relevant PIM policy
- For the Slack notification on `eo pim`: `Get` permission on the Key Vault secret holding the webhook URL (see [Slack notifications](#slack-notifications))

## Usage

```
eo <command> [flags] [arguments]
```

### `eo pim`

Activate the Contributor role for 8 hours. This creates a PIM request and posts
it to Slack, so an approver knows there is something waiting:

```bash
eo pim
eo pim "deploying release 2.4"
```

> :lock: **Bob Mulder** (bob.mulder@eo.nl) requests the **Contributor** role on **EO Studio Digitaal** for 8h.
>
> **Reason**
> deploying release 2.4
>
> <sub>Approve with `eo pim approve` or in [Privileged Identity Management](https://entra.microsoft.com/#view/Microsoft_Azure_PIMCommon/ApprovalMenuBlade/~/azurerbac)</sub>

If the role is already active no request is made, so nothing is posted.

### `eo pim status`

Check whether your Contributor role is currently active and how long it has left.

```bash
eo pim status
```

### `eo pim approve`

List pending PIM requests and approve them.

```bash
# Interactive — pick which requests to approve
eo pim approve

# Approve everything at once
eo pim approve --all

# With a custom justification
eo pim approve --all "sprint review batch"
```

### `eo whoami`

Show the current Azure user (name, email) and active subscription.

```bash
eo whoami
```

### `eo completion`

Output a shell completion script so Tab autocompletes commands.

**zsh** — add to `~/.zshrc`:

```zsh
source <(eo completion zsh)
```

**bash** — add to `~/.bashrc`:

```bash
source <(eo completion bash)
```

**fish** — save to Fish's completions directory:

```fish
mkdir -p ~/.config/fish/completions
eo completion fish > ~/.config/fish/completions/eo.fish
```

After reloading your shell, Tab completion works:

```
eo p<Tab>        → eo pim
eo pim <Tab>     → eo pim approve / eo pim status
```

### `eo version`

Print the current version.

```bash
eo version
```

### Help

```bash
eo --help
eo pim --help
eo pim approve --help
```

## Slack notifications

`eo pim` posts each new request to a Slack [incoming webhook](https://api.slack.com/messaging/webhooks).

The webhook URL is a secret and this repository is public, so it is neither
committed nor baked into the released binaries. Instead the CLI reads it from
Key Vault using the Azure credentials you are already signed in with. The vault
and secret names are constants in
[`internal/pim/notify.go`](./internal/pim/notify.go), and `eo pim --help`
prints them.

Sending a notification needs read access to that secret — `Key Vault Secrets
User` on an RBAC vault, or a `get` secret permission on an access-policy vault.
Check which model the vault uses before granting:

```bash
az keyvault show --name <vault> --query properties.enableRbacAuthorization
```

The notification is best effort: without access to the secret, or when Slack is
unreachable, `eo pim` still activates the role and only reports that the
notification was skipped. It never fails the command.

To move the notifications to another channel, replace the secret rather than
changing the code:

```bash
az keyvault secret set --vault-name <vault> --name <secret> --value <webhook-url>
```

## Releasing a new version

1. Make sure all changes are on `main`
2. Tag the new version and push:

```bash
git tag v1.2.3
git push origin v1.2.3
```

GitHub Actions will automatically build binaries for macOS (Intel + ARM) and Linux, publish a GitHub Release, and update `Casks/eo.rb` for Homebrew installations.
