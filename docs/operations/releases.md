# Releasing eo

1. Merge the release changes into `main`.
2. In GitHub Actions, run **Release** from `main` and enter a tag such as `v1.2.3`. The workflow runs `make check`, creates the tag, uploads the archives, checksums and generated Cask to a draft, and publishes the complete release.
3. Use the compare link in the run summary to open the Cask PR. The Cask job checks the Cask against the published checksums before pushing its branch. Merge that PR after its reviews. Open a Cask PR for the latest release only.
4. Verify the release with Homebrew 7.0.11 or later and an EO GitHub account with repository access. On a clean machine, trust the Cask before running `brew fetch --cask evangelischeomroep/eo-cli/eo`. Run `brew upgrade --cask eo` from an existing installation.

## Retrying a release

- If the release is still a draft, rerun the original workflow. GoReleaser replaces the incomplete draft for that tag.
- Once the release is published, rerun the failed Cask job or dispatch **Release** from `main` with the same tag. It downloads the Cask from the published release and reuses an existing cask branch if the generated Cask matches.

## Protecting release tags

Create a GitHub App for releases with Contents write access and install it on `eo-cli`. In the `release` Actions environment, allow deployments from the `main` branch. Set the App client ID as `RELEASE_APP_CLIENT_ID` and its private key as the `RELEASE_APP_PRIVATE_KEY` secret in that environment.

After confirming the App can create tags, enable a repository tag ruleset for `v*` that restricts creation and grants bypass only to the release App. GitHub can run workflows from old tagged commits, so the tag ruleset is what enforces this restriction outside the workflow. Before the App is configured, **Release** uses the repository workflow token and tag creation follows the current repository permissions.

The source, GitHub Release, and Homebrew tap share one repository. Git authenticates the tap clone and updates. The cask resolves downloads through the GitHub Releases API when Homebrew has GitHub credentials. See the [installation instructions](../../README.md#installation) for login and token setup.

## Switching the repository to private

1. Check that EO users have Homebrew 7.0.11 or later, repository read access and GitHub authentication for both Git and Homebrew.
2. Publish and merge a cask generated with the authenticated download strategy while the repository is still public. This lets existing Homebrew installations pick up the new cask before the visibility change. If the next release must already be private, schedule a window for new installs and upgrades while that release and its cask PR are published.
3. Change visibility after approval. The repository, release URLs and tap name use the same paths.
4. Fetch the cask again with an EO account to verify Git and release asset access. The current CLI uses GitHub CLI credentials or `HOMEBREW_GITHUB_API_TOKEN` for its update check.
