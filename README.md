# bipolar

> Use a different SSH key per GitHub org — automatically, with zero friction.

If you juggle multiple GitHub accounts (personal, work, enterprise, client), you know the pain: wrong SSH key, `Permission denied`, stalled deploys. **bipolar** fixes that by transparently swapping your SSH identity based on the org in the remote URL — no wrappers, no per-repo config, no thinking required.

![downloadbipolaq](https://github.com/user-attachments/assets/7596e167-25bf-4de6-9d57-36c3fa8d59b1)

---

## How it works

bipolar installs a shell function (`git-dispatch`) that wraps the `git` command. On any network operation (`push`, `pull`, `fetch`, `clone`), it:

1. Reads the remote URL
2. Extracts the org/username
3. Matches it against your configured profiles (first match wins)
4. Injects `GIT_SSH_COMMAND` with the right key for that profile
5. Falls back to your system SSH defaults if nothing matches

It's a pure shell function — no daemon, no background process, no latency.

---

## No-install method (web) ⚡

Don't want to download anything? **You don't have to.** The binary only ever did two things — drop a shell function into your rc file and write a config file — so we put that in the browser.

→ **[Open the web installer](https://scicom-ai-enterprise-organization.github.io/git-bipolar/)**

1. Add your profiles in the form — org/username pattern, SSH key path, optional commit identity and signing.
2. Copy the generated **one-liner** into your terminal (or download `install.sh` and read every line first).
3. Reload your shell: `source ~/.zshrc` (or `~/.bashrc`).

Done — every `git push/pull/fetch/clone` is now intercepted, exactly as if you'd used the binary.

**It's safe by construction.** The page runs entirely client-side and talks to no server. The command it generates only writes file *paths* into `~/.git_profiles.conf` and appends the managed `git-dispatch` block to your rc file — it never reads, uploads, or touches the contents of your SSH keys. Everything it will do is shown in the previews on the page before you run anything.

Your profiles are also **saved locally in your browser** (via `localStorage`), so you can close the tab and come back later to tweak them — nothing is sent anywhere, and a one-click **Clear saved data** wipes that copy.

The block it installs is **byte-for-byte identical** to the one the binary ships (CI enforces this), so the two are interchangeable: install via the web today, manage with the binary later, or never touch the binary at all.

> Requires `bash`/`zsh` and `openssl` (preinstalled on macOS and most Linux). On Windows, use Git Bash or WSL.

The binary below is still handy if you prefer an **interactive menu**, a built-in SSH-key file browser, and the **Repair** diagnostic — but it's entirely optional.

---

## Installation

### macOS

**Apple Silicon (M1/M2/M3):**
```sh
curl -L https://github.com/Scicom-AI-Enterprise-Organization/git-bipolar/releases/latest/download/bipolar-darwin-arm64 -o bipolar
chmod +x bipolar
sudo mv bipolar /usr/local/bin/bipolar
```

**Intel:**
```sh
curl -L https://github.com/Scicom-AI-Enterprise-Organization/git-bipolar/releases/latest/download/bipolar-darwin-amd64 -o bipolar
chmod +x bipolar
sudo mv bipolar /usr/local/bin/bipolar
```

Verify:
```sh
bipolar --version
```

---

### Linux

**x86_64:**
```sh
curl -L https://github.com/Scicom-AI-Enterprise-Organization/git-bipolar/releases/latest/download/bipolar-linux-amd64 -o bipolar
chmod +x bipolar
sudo mv bipolar /usr/local/bin/bipolar
```

**ARM64:**
```sh
curl -L https://github.com/Scicom-AI-Enterprise-Organization/git-bipolar/releases/latest/download/bipolar-linux-arm64 -o bipolar
chmod +x bipolar
sudo mv bipolar /usr/local/bin/bipolar
```

Verify:
```sh
bipolar --version
```

---

### Windows

> **Requires Git Bash or WSL.** Native PowerShell / CMD are not supported — the shell function that bipolar installs is bash/zsh only.

1. Download `bipolar-windows-amd64.exe` from the [Releases](../../releases) page
2. Rename it to `bipolar.exe` and move it somewhere on your `PATH`, e.g. `C:\Program Files\bipolar\bipolar.exe`
3. Open **Git Bash** and run `bipolar`

---

### Build from source

Requires Go 1.23+.

```sh
git clone https://github.com/Scicom-AI-Enterprise-Organization/git-bipolar
cd bipolar
make build
# binary at bin/bipolar
sudo mv bin/bipolar /usr/local/bin/bipolar
```

To cross-compile all platforms at once:

```sh
make build-all    # outputs to dist/
make release      # same, but also produces .tar.gz / .zip archives
```

---

## First run

Just run `bipolar`. On first launch it will:

1. Detect your shell (zsh, bash, etc.) and its rc file
2. Install the `git-dispatch` function into that rc file, wrapped in a managed block
3. Create an empty `~/.git_profiles.conf`

Then source your rc file to activate it in the current session:

```sh
source ~/.zshrc   # or ~/.bashrc
```

After that, every `git push/pull/fetch/clone` is automatically intercepted.

---

## Managing profiles

Run `bipolar` to open the interactive menu:

```
? Bipolar  —  Git SSH Profile Manager
  > Configure Profiles
    Repair Bipolar
    Exit
```

### Adding a profile

Select **Configure Profiles → + Add new profile** and fill in three fields:

| Field | Description |
|-------|-------------|
| **Profile name** | A short label, e.g. `Work` or `PersonalGH` |
| **Org / username pattern** | Plain string or regex matching the GitHub org or username |
| **SSH key file** | Navigate to your private key with the built-in file browser |
| **Git user.name / user.email** *(optional)* | Commit identity for this profile. Injected with `git -c …`, which overrides both local `.git/config` and global `~/.gitconfig`. Leave blank to keep your global identity. |
| **Commit signing** *(optional, opt-in)* | Sign commits and tags with an SSH key. Defaults to your profile's SSH key, or pick a different one. See [Commit signing](#commit-signing). |

The pattern is matched against the org name extracted from the remote URL. Examples:

```
acme-corp          → exact match
^(acme|widgets)$   → matches either org
.*                 → matches everything (use as a last resort)
```

Profiles are matched **top to bottom, first match wins**. Unmatched repos fall through to your system SSH defaults.

### Config file

Profiles are stored in `~/.git_profiles.conf` in a simple INI format:

```ini
[Work]
key_file=~/.ssh/id_ed25519_work
git_name=Jane Work
git_email=jane@acme-corp.com
sign=true
sign_key=~/.ssh/id_ed25519_work
org_pattern=acme-corp

[PersonalGH]
key_file=~/.ssh/id_ed25519_personal
org_pattern=my-github-username
```

`git_name`, `git_email`, `sign`, and `sign_key` are all optional. You can edit this file directly — bipolar will pick up changes immediately (no reload needed).

---

## Commit signing

Each profile can optionally **sign commits and tags** using [git's SSH signing](https://git-scm.com/docs/git-config#Documentation/git-config.txt-gpgformat) (requires git **2.34+**). It's **opt-in** — disabled unless you enable it on a profile.

Enable it when adding or editing a profile:

```
? Enable commit signing for this profile? Yes
? Sign with this profile's SSH key? (~/.ssh/id_ed25519_work) Yes
```

Accept the default to sign with the profile's own SSH key, or choose **No** to pick a dedicated signing key from the file browser. Signed profiles are marked with a 🔏 in the menu.

When a signed profile matches, bipolar injects the signing config for that one command and prints:

```
[Git-Bipolar] 🔏 Signing commit with ~/.ssh/id_ed25519_work
```

Under the hood it runs the equivalent of:

```sh
git -c gpg.format=ssh \
    -c user.signingkey=~/.ssh/id_ed25519_work \
    -c commit.gpgsign=true \
    -c tag.gpgsign=true \
    commit ...
```

> To **verify** signatures locally (e.g. `git log --show-signature`), add the key to a `~/.ssh/allowed_signers` file and set `gpg.ssh.allowedSignersFile`. This is only needed for verification — signing works without it.

---

## Repair Bipolar

**Repair Bipolar** in the menu runs a diagnostic check:

- Verifies the `git-dispatch` block exists in your rc file
- Verifies the block matches the current version (catches stale installs after upgrades)
- Verifies `~/.git_profiles.conf` exists and all profiles are syntactically valid
- Offers to fix anything that's broken

The managed block in your rc file is clearly delimited so it's easy to find and won't interfere with your other config:

```sh
### Managed by Bipolar ###
git-dispatch() { ... }
alias git='git-dispatch'
### Bipolar End ###
```

---

## Windows note

The `git-dispatch` shell function requires bash or zsh. On Windows, bipolar works in **Git Bash**, **WSL**, or any bash-compatible environment. Native PowerShell / CMD are not supported.

---

## Releasing

Releases are automated via GitHub Actions. The workflow builds all platform binaries, packages them as archives, and publishes a GitHub Release with auto-generated notes.

### Versioning

bipolar follows [Semantic Versioning](https://semver.org/):

```
v<MAJOR>.<MINOR>.<PATCH>

MAJOR — breaking change (e.g. config format change, removed feature)
MINOR — new feature, backwards compatible
PATCH — bug fix, backwards compatible
```

### Cutting a release

Use `make tag` — it validates your environment before pushing anything:

```sh
make tag VERSION=v1.2.3
```

This will:
1. Verify `VERSION` is set and follows semver
2. Verify the working tree is clean (no uncommitted changes)
3. Verify the tag doesn't already exist
4. Create an annotated git tag and push it to origin
5. Print a link to watch the Actions run

The GitHub Actions [release workflow](.github/workflows/release.yml) then kicks in automatically and:
- Cross-compiles for all 5 targets (darwin amd64/arm64, linux amd64/arm64, windows amd64)
- Packages `.tar.gz` archives for mac/linux and a `.zip` for windows
- Creates a GitHub Release with the archives attached and auto-generated release notes

### If something goes wrong

Delete the tag locally and remotely, fix the issue, then re-tag:

```sh
git tag -d v1.2.3
git push origin :refs/tags/v1.2.3
# fix things...
make tag VERSION=v1.2.3
```

---

## Contributing

PRs welcome. The codebase is small and straightforward:

```
cmd/bipolar/main.go            entry point
internal/shell/shell.go        shell detection, rc file management, embedded function
internal/profiles/profiles.go  config file parse / save / validate
internal/ui/menu.go            interactive menus
internal/ui/filepicker.go      SSH key file browser
docs/index.html                no-install web installer (client-side, self-contained)
scripts/sync-web.mjs           regenerates the block embedded in docs/index.html
```

```sh
make tidy      # go mod tidy
make lint      # go vet
make build     # compile for current platform
make sync-web  # re-embed the shell block into the web installer
make check-web # verify the web installer is in sync (used by CI)
```

### Keeping the web installer in sync

`docs/index.html` embeds the canonical `git-dispatch` block from
`internal/shell/shell.go`. **If you change that block, run `make sync-web`** and
commit the updated `docs/index.html`. CI runs `make check-web` and fails if the
two have drifted, so the web installer and the binary always emit identical code.

### Deploying the web installer

The site is served from the `docs/` folder by GitHub Pages via
[`.github/workflows/pages.yml`](.github/workflows/pages.yml). To enable it once:
**Settings → Pages → Build and deployment → Source: GitHub Actions.** After that,
every push to `main` that touches `docs/` redeploys automatically to
`https://scicom-ai-enterprise-organization.github.io/git-bipolar/`.

---

## License

MIT
