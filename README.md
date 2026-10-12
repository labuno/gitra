# gitra

[English](README.md) | [简体中文](README.zh-CN.md)

**The right Git account for every folder.** A small Git identity manager with a CLI and a card-based terminal UI.

> Bind once, Git forever: bind a repository to an account once, then use plain
> `git pull / push / fetch`, your editor, and your scripts with the right
> identity — no account switching, no daemon, no Git protocol implementation.

---

## What it solves

Using work, personal, and open-source Git accounts on the same computer can lead to conflicting defaults:

- **Commit identity:** global `user.name` and `user.email` settings can put your personal email on a work commit.
- **Credentials:** a shared credential entry for `github.com` can select the wrong account, causing `git pull` to fail with `Repository not found`.
- **SSH keys:** using several keys for one host often requires SSH host aliases, rewritten remote URLs, or repeated SSH agent changes.

gitra makes “which account does this folder use?” a one-time binding. Each repository gets its own **commit identity and credential scope**, so you can keep using Git as usual.

## Who it is for

- Developers using **two or more Git identities on one computer**: work and personal accounts, multiple GitHub accounts, or GitHub alongside GitLab.
- Anyone who wants to avoid configuring `includeIf`, `core.sshCommand`, SSH host aliases, and credential helpers by hand.
- Users who prefer signing in and binding folders through a terminal UI launched with a double-click.
- Developers who want editors, AI coding tools, scripts, and local automation to use the appropriate identity for each repository.

If you only use one Git account, the standard provider CLI or a credential helper may already meet your needs.

## How it works

```text
Sign in once → Bind each project folder → Repository-local configuration:
                                          user.name / user.email
                                          credential.<url>.username or SSH key command
                                          gitra metadata + pre-binding snapshot
                                                       ↓
                                          Plain Git, editors, and scripts
```

- Credentials live in the **system credential store**: macOS Keychain, Windows Credential Manager, or Linux Secret Service. Tokens are not written into repository configuration.
- gitra saves the original configuration before binding; `unbind` restores it.
- It does not rewrite existing remote URLs, change your global `~/.gitconfig` or `~/.ssh/config`, or run a background daemon.

Supported providers: **GitHub, GitLab, and Gitea**, including self-hosted instances with custom hosts and SSH ports.
Authentication: HTTPS credentials by default, or SSH keys for advanced use.

## Getting started

### macOS: double-click to launch

1. Open this repository's **Releases** page and download `Gitra-macOS.zip`.
2. Extract the archive and double-click `Gitra.app`. A Terminal window opens with the gitra interface.
3. If macOS blocks the first launch, open **System Settings → Privacy & Security → Open Anyway**, then launch the app again. The archive also includes first-launch instructions.
4. Choose **Sign in to GitHub** in the welcome menu. The terminal UI currently uses Chinese labels:
   - Use the official `gh` CLI for browser authorization or reuse a signed-in account.
   - Alternatively, paste an access token; gitra can open the provider's token page with the required permissions selected.
5. Choose a project folder and bind it to the account. If the HTTPS remote does not exist, gitra offers to create it. Once you have a local commit, press `U` for the first upload; afterward, sync through your editor or Git.

Keyboard controls: arrow keys to move, `Enter` to confirm, `Esc` to go back, `A` to add an account, `B` to bind a project, `T` to test an account, `U` for the first upload, and `Q` to quit after confirmation.

You can select a regular project folder without initializing Git beforehand. gitra initializes a `main` branch and guides you through remote setup. HTTPS accounts can create a missing repository; SSH accounts associate an SSH remote address. Initialization does not stage, commit, or upload files. If you cancel remote setup, the local Git repository remains unbound. Existing history and remote URLs are preserved.

In the folder picker, press `S` to save the current folder as the starting directory. This preference is stored in `picker.json` under the gitra configuration directory and applies across accounts and restarts. Without a saved preference, gitra uses its existing defaults and session memory. If the saved directory is deleted, correct or remove `picker.json` before reopening the picker.

> To run from source, use `./scripts/build-app.sh` to generate `dist/Gitra.app`,
> or double-click `Gitra.command` at the repository root. The latter builds on first launch and requires Go.

### CLI: scripts and automation

The CLI `bind` command expects an initialized Git repository; automatic folder initialization is available in the terminal UI's single-folder binding flow.

```bash
gitra login --provider github         # Browser authorization or gh reuse
gitra account list                    # List accounts; supports --json
gitra bind <alias> [path]             # Bind the current or specified repository
gitra status [path]                   # Inspect binding health; supports --json
gitra account test <alias>            # Explicit online identity verification
gitra publish [path]                  # One-time first upload
gitra unbind [path]                   # Restore the pre-binding configuration
```

Exit codes: `0` success, `2` arguments or validation, `3` account errors, `4` repository errors, `5` authentication errors, `6` conflicts or configuration drift, `7` unsupported operations, and `1` other errors.

## Credentials and privacy

- gitra has no hosted account system, telemetry, or cloud backend.
- Credentials stay in your local system credential store. Tokens are not written into gitra's JSON configuration, logs, or repository configuration.
- **Git itself** retrieves credentials from the system store. Repository authentication does not require a running gitra process.
- `gitra logout <alias>` deletes the stored credential, clears credential scoping from bound repositories, and provides the provider's revocation URL.

## Scope

gitra manages repository identity and authentication. It does not provide a Git GUI, display history, branches, pull requests, or issues, or proxy routine Git commands. There are no `gitra push`, `pull`, or `fetch` commands. The one-time `gitra publish` operation pushes your existing local branch and establishes its upstream. gitra does not rewrite existing remote URLs or run a daemon.

## Current status and limitations

- Implemented: account management, HTTPS and SSH binding and restoration, login through provider CLIs or access tokens, explicit identity verification, first upload, a card-based terminal UI, single-folder Git initialization, saved picker starting directories, JSON output, and defined exit codes. The full `go test ./...` suite and `go vet ./...` pass.
- Planned: built-in device flow, account-field editing and a clone wizard in the terminal UI, and first-class Windows support. Built-in device flow requires a registered OAuth application's client ID.
- Passphrase-protected SSH keys must be loaded into the SSH agent. The SSH login picker can invoke `ssh-add`; identity probes do not prompt for a passphrase.
- Automatic initialization applies to single-folder binding in the terminal UI. CLI and bulk binding expect existing Git repositories.

## Releases and automation

- **CI** (`.github/workflows/ci.yml`): pushes and pull requests targeting `main` run formatting checks, vet, tests, and builds, plus a macOS application-bundle smoke check.
- **Release** (`.github/workflows/release.yml`): pushing a tag beginning with `v` builds and publishes release assets:

  ```bash
  git tag v0.1.0          # For example: v0.1.0 or v1.2.3-rc1
  git push origin v0.1.0  # Trigger the release workflow
  ```

  New releases upload only the universal macOS application archive `Gitra-macOS.zip` and `checksums.txt`. GitHub also provides two source archives automatically.
  Earlier releases may include separate CLI archives for macOS, Linux, and Windows; those existing downloads are retained. Build the CLI from source when needed.
  The workflow creates the release if needed and attaches assets to the matching tag.

The binary includes its version: `gitra --version` prints the release tag, or `dev` for a local build.

For release candidates, use tags such as `v0.1.4-rc1`, followed by `v0.1.4-rc2` if fixes are needed.
They run the same release workflow and publish the same assets, marked as **Pre-release** without replacing the latest stable release.
After verifying a candidate, publish a stable tag such as `v0.1.4` at the verified commit. Create a new tag for each candidate instead of moving existing tags.

Before tagging a release, add `docs/releases/<tag>.md` with English and Chinese notes covering new features, fixes, limitations, and downloads. The workflow uses that document as the release body and stops if it is missing or empty.

## First push

Create a local commit before uploading a new repository. The first push sets the upstream branch:

```bash
git push -u origin main  # Once; alternatively use U in gitra or gitra publish
```

Afterward, use your editor or plain `git push` and `git pull`.

## Development

Requires Go 1.25.1 or later and the native Git executable. Building the macOS app also requires macOS tools such as `lipo` and `codesign`.

```bash
go build ./... && go test ./... && go vet ./...
./scripts/build-app.sh  # Generate dist/Gitra.app
```

Architecture and behavioral contracts are documented in `docs/`:

- [Architecture and development design](docs/gitra-architecture-development-design.md)
- [V1 implementation baseline](docs/gitra-v1-implementation-baseline.md)
- [Login and onboarding design](docs/gitra-v1-login-onboarding-design.md)
- Implementation plans and verification evidence under `docs/plans/` and `docs/todos/`

These detailed design documents are currently in Chinese.
