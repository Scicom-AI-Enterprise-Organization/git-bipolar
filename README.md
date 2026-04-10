# bipolar

> Use a different SSH key per GitHub org — automatically, with zero friction.

If you juggle multiple GitHub accounts (personal, work, enterprise, client), you know the pain: wrong SSH key, `Permission denied`, stalled deploys. **bipolar** fixes that by transparently swapping your SSH identity based on the org in the remote URL — no wrappers, no per-repo config, no thinking required.

```
[Git-Bipolar] Org: acme-corp | Using Work Profile...
```

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

## Installation

### macOS

**Apple Silicon (M1/M2/M3):**
```sh
curl -L https://github.com/your-org/bipolar/releases/latest/download/bipolar-darwin-arm64 -o bipolar
chmod +x bipolar
sudo mv bipolar /usr/local/bin/bipolar
```

**Intel:**
```sh
curl -L https://github.com/your-org/bipolar/releases/latest/download/bipolar-darwin-amd64 -o bipolar
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
curl -L https://github.com/your-org/bipolar/releases/latest/download/bipolar-linux-amd64 -o bipolar
chmod +x bipolar
sudo mv bipolar /usr/local/bin/bipolar
```

**ARM64:**
```sh
curl -L https://github.com/your-org/bipolar/releases/latest/download/bipolar-linux-arm64 -o bipolar
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
git clone https://github.com/your-org/bipolar
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
org_pattern=acme-corp

[PersonalGH]
key_file=~/.ssh/id_ed25519_personal
org_pattern=my-github-username
```

You can edit this file directly — bipolar will pick up changes immediately (no reload needed).

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
cmd/bipolar/main.go          entry point
internal/shell/shell.go      shell detection, rc file management, embedded function
internal/profiles/profiles.go  config file parse / save / validate
internal/ui/menu.go          interactive menus
internal/ui/filepicker.go    SSH key file browser
```

```sh
make tidy   # go mod tidy
make lint   # go vet
make build  # compile for current platform
```

---

## License

MIT
