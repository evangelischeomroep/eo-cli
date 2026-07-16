# eo-cli

Developer CLI for Evangelische Omroep. Currently focused on Azure PIM: activating your own role and approving requests from teammates.

## Installation

### macOS (recommended)

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

## Usage

```
eo <command> [flags] [arguments]
```

### `eo pim`

Activate the Contributor role for 8 hours.

```bash
eo pim
eo pim "deploying release 2.4"
```

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

After reloading your shell (`source ~/.zshrc` or open a new terminal), Tab completion works:

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

## Releasing a new version

1. Make sure all changes are on `main`
2. Tag the new version and push:

```bash
git tag v1.2.3
git push origin v1.2.3
```

GitHub Actions will automatically build binaries for macOS (Intel + ARM) and Linux, publish a GitHub Release, and update `Casks/eo.rb` for Homebrew installations.
