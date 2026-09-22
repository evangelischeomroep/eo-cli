# eo-cli

Developer CLI for Evangelische Omroep. Azure PIM (activating your own role, approving requests from teammates) and EOchat, the EO AI assistant, straight from your terminal.

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
- For `ask` / `mcp`: an EOchat account on [chat.eo.nl](https://chat.eo.nl) and a personal API key (see below)

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

### `eo ask`

Ask [EOchat](https://chat.eo.nl) (Open WebUI, with EO's custom models and knowledge bases) from the terminal. Answers stream in as they are generated.

**First time:** create a personal API key in EOchat under _Settings → Account → API keys_, then:

```bash
eo ask login
```

The key is checked, stored in `~/.config/eo/eochat.json` (mode 0600), and you pick a default model. Alternatively set `EOCHAT_API_KEY` in your environment.

```bash
# One question
eo ask "wat is het verschil tussen een Function App en een Container App?"

# Interactive chat (Claude Code style): type, get streamed answers, /help for commands
eo ask

# Pipe anything in as context
git diff | eo ask "review this change, focus on bugs"
make build 2>&1 | eo ask "what went wrong?"
cat error.log | eo ask

# Continue the previous conversation
eo ask -c "and how do I test that?"

# Pick a model for this question (ID, name, or a unique part of it)
eo ask -m gpt "..."

# Ground the answer in an EOchat knowledge base, or in a file you upload
eo ask -k "EO Handboek" "hoe vraag ik PIM aan?"
eo ask -f ./architectuur.pdf "summarize the trade-offs"

# System prompt, web search, plain output for scripts
eo ask -s "answer in Dutch, be brief" "..."
eo ask --web "..."
eo ask --raw "..." > answer.md
```

Housekeeping:

```bash
eo ask status          # server, user, default model, last conversation
eo ask models          # list the models you can use
eo ask model           # choose a default model (or: eo ask model <name>)
eo ask knowledge       # list knowledge bases usable with -k
eo ask logout          # remove the stored key
```

In the interactive chat: `/model [name]`, `/models`, `/knowledge`, `/use <kb>`, `/file <path>`, `/system <text>`, `/web`, `/new`, `/quit`. End a line with `\` to continue on the next line. Ctrl+C stops the current answer; Ctrl+D quits.

Conversations are kept locally (for `-c`), not in the EOchat web history.

### `eo mcp`

Let Claude Code (or any MCP client) use EOchat as a tool. `eo mcp` runs a [Model Context Protocol](https://modelcontextprotocol.io) server on stdin/stdout that exposes `eochat_ask`, `eochat_list_models` and `eochat_list_knowledge`, using the same login as `eo ask`.

```bash
eo ask login          # once, if you have not already
eo mcp install        # runs: claude mcp add --scope user eochat -- eo mcp
```

Use `eo mcp install --project` to register it in the current repository's `.mcp.json` instead. Manual configuration for other clients:

```json
{ "mcpServers": { "eochat": { "command": "eo", "args": ["mcp"] } } }
```

Then, inside Claude Code: _"Use eochat_ask to check how EO names Azure resources"_ or _"Ask EOchat's EO Handboek knowledge base how PIM approvals work"_.

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
eo ask <Tab>     → eo ask login / models / model / ...
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
