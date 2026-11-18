# Releasing eo

1. Merge the release changes into `main`.
2. In GitHub Actions, run **Release** from `main` and enter a tag such as `v1.2.3`. The workflow runs `make check` before pushing the tag. A release tag pushed manually also starts the workflow if its commit is on `main`.
3. Use the compare link in the run summary to open the cask PR. GoReleaser pushes the cask branch and the organization limits PR creation by workflow tokens. Once the PR has the usual reviews and merges, Homebrew serves the new version.
4. Verify the release with `brew fetch --cask evangelischeomroep/eo-cli/eo` and `brew upgrade --cask eo` using a GitHub account with repository access.

The source, GitHub Release, and Homebrew tap share one repository. Git authenticates the tap clone and updates. The cask resolves downloads through the GitHub Releases API when Homebrew has GitHub credentials. See the [installation instructions](../../README.md#installation) for login and token setup.

## Switching the repository to private

1. Check that EO users have read access and have configured GitHub authentication for both Git and Homebrew.
2. Publish and merge a cask generated with the authenticated download strategy while the repository is still public. This lets existing Homebrew installations pick up the new cask before the visibility change. If the next release must already be private, schedule a window for new installs and upgrades while that release and its cask PR are published.
3. Change visibility after approval. The repository, release URLs and tap name use the same paths.
4. Fetch the cask again with an EO account to verify Git and release asset access. The current CLI uses GitHub CLI credentials or `HOMEBREW_GITHUB_API_TOKEN` for its update check.
