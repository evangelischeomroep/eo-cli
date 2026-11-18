# eo-cli

`eo` is a single Go binary that EO developers install through Homebrew to use Azure PIM and EOchat from the terminal. The repo is public, and every `v*` tag ships a release to everyone who installed it.

## What we never compromise on

1. **A production deploy stays gated.** Any new production deploy path must verify that each app's test stage has completed and get the user's confirmation of the app list before approving deployment.
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

It runs `go mod tidy`, the tests, `go vet` and a build. Commit any changes `tidy` makes to `go.mod` and `go.sum`.

Use the [pull-request skill](.agents/skills/pull-request/SKILL.md) when opening or updating a PR. Apply the `warden` label to request a review on a non-draft PR from this repository.

## Releasing

Run **Release** from `main` in GitHub Actions and enter a `vX.Y.Z` tag. The workflow checks the code, pushes the tag, publishes the release and links to the generated Homebrew cask update. Homebrew serves the new version after the cask PR merges. See the [release runbook](docs/operations/releases.md) for the steps and the private repository cutover.

Commits starting with `chore:`, `ci:` or `docs:` stay out of the changelog.

## Documentation

Most changes need no documentation; agents can read the code.

- The README is the user manual of a public tool: a new command or flag gets its usage there.
- `docs/internals/` holds decisions, their reasons, and constraints that span components. Add a paragraph only when a maintainer would get something wrong without it.
- `docs/operations/` holds runbooks for setup, deploys and debugging.
- A reason that belongs to one spot in the code goes in a short comment there.
- When a documented decision changes, rewrite the text; a doc describes the current state.

## How it works

`cmd/eo/main.go` assembles the Cobra command tree. The Azure commands get their tokens from the Azure CLI (`az account get-access-token`) and call Azure Resource Manager and Microsoft Graph directly. `eo ask` and `eo mcp` talk to EOchat, an Open WebUI server, with the user's API key. Once a day the binary checks GitHub for a newer release and prints a notice.

## Where code lives

- `cmd/eo` - the commands, their help texts and the shell completion scripts.
- `internal/azure` - the `az` token calls and the Azure REST client.
- `internal/pim` - PIM activation, status and approvals.
- `internal/eochat` - the EOchat client, the stored config and local conversations.
- `internal/mcp` - the MCP server behind `eo mcp`.

## Taste

- **Define commands and flags once.** Cobra generates help and shell completions from the command definitions. Add usage for new commands and flags to the README.
