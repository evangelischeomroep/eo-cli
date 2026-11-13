# eo-cli

`eo` is a single Go binary that EO developers install through Homebrew to run Azure PIM, Function App deploys and EOchat from the terminal. The repo is public, and every `v*` tag ships a release to everyone who installed it.

## What we never compromise on

1. **A production deploy stays gated.** `eo deploy prod` only approves apps whose test stage has completed, and only after the user confirms the list. Keep both checks for every new deploy path.
2. **The EOchat key stays private.** The key is written to the user's config with mode 0600, and `EOCHAT_API_KEY` overrides it. Never log it or write it anywhere else.

## Ways to hurt yourself

1. **Tagging from a branch.** Pushing a `v*` tag builds and publishes a release, and every installed `eo` starts announcing it within a day. Tag only a commit that is on `main`.
2. **Editing the Homebrew cask by hand.** GoReleaser generates `Casks/eo.rb` on every release, so a hand edit is lost. Change the cask through `homebrew_casks` in `.goreleaser.yaml`.

## Running locally

`go run ./cmd/eo <command>` runs the CLI from source; `make build` writes a binary to `/tmp/eo`. A local build has version `dev`, which skips the update check. The Azure commands need `az login` and access to the _EO Studio Digitaal_ subscription; `pim approve` also needs you to be an approver on the PIM policy.

| Name             | Purpose                                               |
| ---------------- | ----------------------------------------------------- |
| `EOCHAT_API_KEY` | EOchat API key; overrides the one from `eo ask login` |
| `EOCHAT_MODEL`   | Default model; overrides the stored one               |
| `EOCHAT_URL`     | EOchat server; defaults to `https://chat.eo.nl`       |
| `NO_COLOR`       | Turns off coloured output                             |

## Done means green

A change is done when this passes:

```bash
make check
```

It runs `go mod tidy`, the tests, `go vet` and a build; commit any changes `tidy` makes to `go.mod` and `go.sum`. No workflow runs on pull requests, so this is the only gate before a release.

## Releasing

Tag the release on `main` and push the tag:

```bash
git tag v1.2.3
git push origin v1.2.3
```

The Release workflow runs GoReleaser: it builds binaries for macOS and Linux (Intel and ARM), publishes the GitHub Release, and opens a pull request that updates the Homebrew cask, because `main` only accepts pull requests. Merge that pull request; until then, Homebrew users stay on the previous version. Commits starting with `chore:`, `ci:` or `docs:` stay out of the changelog.

## Documentation

Most changes need no documentation; agents can read the code.

- The README is the user manual of a public tool: a new command or flag gets its usage there.
- `docs/internals/` holds decisions, their reasons, and constraints that span components. Add a paragraph only when a maintainer would get something wrong without it.
- `docs/operations/` holds runbooks for setup, deploys and debugging.
- A reason that belongs to one spot in the code goes in a short comment there.
- When a documented decision changes, rewrite the text; a doc describes the current state.

## How it works

`cmd/eo/main.go` dispatches each command. The Azure commands get their tokens from the Azure CLI (`az account get-access-token`) and call Azure Resource Manager, Microsoft Graph and Azure DevOps directly. `eo deploy` lists the Function Apps in `rg-<env>-webapps` and runs the Azure DevOps pipeline named after the app, without its `app-<env>-` prefix, plus `_release`. `eo ask` and `eo mcp` talk to EOchat, an Open WebUI server, with the user's API key. Once a day the binary checks GitHub for a newer release and prints a notice.

## Where code lives

- `cmd/eo` - the commands, their help texts and the shell completion scripts.
- `cmd/eo/pipeline-map.json` - Function Apps whose deploy pipeline has a different name. It is embedded in the binary, so a new entry needs a release.
- `internal/azure` - the `az` token calls and the Azure REST client.
- `internal/pim` - PIM activation, status and approvals.
- `internal/deploy` - Function App listing and Azure DevOps pipeline stages.
- `internal/eochat` - the EOchat client, the stored config and local conversations.
- `internal/mcp` - the MCP server behind `eo mcp`.

## Taste

- **A command reaches every surface.** A new command or flag goes into the dispatch in `main.go`, its help text in `help.go`, the zsh, bash and fish scripts in `completion.go`, and the README.
