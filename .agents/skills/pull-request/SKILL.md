---
name: pull-request
description: Use when opening or updating an eo-cli pull request. Check the whole branch, run make check, and write the title and description in EO house style.
---

# Pull requests

Read [WRITING-GUIDE.md](WRITING-GUIDE.md) and the [PR template](../../../.github/pull_request_template.md) before drafting the PR.

1. Fetch the base branch and inspect the status, commits and diff from the merge base. Cover the whole branch in the description. Use the ticket key from the branch name or from the task.
2. Run `make check`. Include tracked dependency updates from `go mod tidy` when they belong to this change. Stage the task's changes and use a topic branch targeting `main`.
3. When the user requests a PR, push the branch and open the PR with `gh pr create --body-file` or the connected PR tool. If the branch already has an open PR, preserve the author's description and review checkbox unless the user asks for an edit.
4. Return the PR URL and report verification to the user. Warden reviews non-draft PRs from this repository when a maintainer applies the `warden` label. If a check is running and the user wants it monitored, use the host's PR watcher.
