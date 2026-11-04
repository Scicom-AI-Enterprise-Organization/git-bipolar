package shell

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestHTTPSProfileAuth(t *testing.T) {
	for _, sh := range []string{"bash", "zsh"} {
		t.Run(sh, func(t *testing.T) {
			if _, err := exec.LookPath(sh); err != nil {
				t.Skip(sh + " unavailable")
			}
			for _, tc := range []struct {
				name, remote, config, command, want string
				fail                                bool
			}{
				{"personal pull", "https://github.com/alice/private.git", "[personal]\ngh_user=alice\ngit_name=Alice\ngit_email=alice@example.com\norg_pattern=^alice$\n", "pull", "TOKEN=selected-test-token", false},
				{"fetch", "https://github.com/alice/private.git", "[personal]\ngh_user=alice\norg_pattern=^alice$\n", "fetch", "TOKEN=selected-test-token", false},
				{"push", "https://github.com/alice/private.git", "[personal]\ngh_user=alice\norg_pattern=^alice$\n", "push", "TOKEN=selected-test-token", false},
				{"ls remote", "https://github.com/alice/private.git", "[personal]\ngh_user=alice\norg_pattern=^alice$\n", "ls-remote origin HEAD", "TOKEN=selected-test-token", false},
				{"SSH key", "git@github.com:alice/private.git", "[personal]\nkey_file=~/.ssh/test-key\ngh_user=alice\norg_pattern=^alice$\n", "fetch", "IdentitiesOnly=yes", false},
				{"clone destination", "", "[personal]\ngh_user=alice\norg_pattern=^alice$\n", "clone https://github.com/alice/private.git nested/checkout", "TOKEN=selected-test-token", false},
				{"no account fallback", "https://github.com/alice/private.git", "[personal]\norg_pattern=^alice$\n", "fetch", "TOKEN=existing-test-token", false},
				{"non GitHub fallback", "https://gitlab.com/alice/private.git", "[personal]\ngh_user=alice\norg_pattern=^alice$\n", "pull", "TOKEN=existing-test-token", false},
				{"local command", "https://github.com/alice/private.git", "[personal]\ngh_user=alice\norg_pattern=^alice$\n", "status", "TOKEN=existing-test-token", false},
				{"Git failure", "https://github.com/alice/private.git", "[personal]\ngh_user=alice\norg_pattern=^alice$\n", "pull", "TOKEN=selected-test-token", true},
				{"missing account", "https://github.com/alice/private.git", "[personal]\ngh_user=missing\norg_pattern=^alice$\n", "pull", "Cannot load GitHub credentials", true},
				{"no account leakage", "https://github.com/bob/private.git", "[personal]\ngh_user=alice\norg_pattern=^alice$\n[work]\norg_pattern=^bob$\n", "pull", "TOKEN=existing-test-token", false},
				{"signed identity", "https://github.com/alice/private.git", "[personal]\ngh_user=alice\ngit_name=Alice\ngit_email=alice@example.com\nsign=true\nsign_key=~/.ssh/sign.pub\norg_pattern=^alice$\n", "pull", "<commit.gpgsign=true>", false},
			} {
				t.Run(tc.name, func(t *testing.T) {
					dir := t.TempDir()
					write := func(name, body string) {
						t.Helper()
						if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0700); err != nil {
							t.Fatal(err)
						}
					}
					write(".git_profiles.conf", tc.config)
					if err := os.Mkdir(filepath.Join(dir, ".ssh"), 0700); err != nil {
						t.Fatal(err)
					}
					write(".ssh/test-key", "test key path only")
					write("git", `#!/bin/sh
if [ "$1" = config ]; then printf '%s\n' "$TEST_REMOTE"; exit; fi
printf 'TOKEN=%s\n' "$GH_TOKEN"
printf 'SSH=%s\n' "$GIT_SSH_COMMAND"
for arg do printf '<%s>\n' "$arg"; done
exit "${TEST_GIT_EXIT:-0}"
`)
					write("gh", `#!/bin/sh
printf '%s\n' "$*" >> "$HOME/gh-calls"
[ -z "$GH_TOKEN" ] && [ -z "$GITHUB_TOKEN" ] || exit 4
case "$*" in
 'auth token --hostname github.com --user alice') printf 'selected-test-token\n';;
 *) exit 1;;
esac
`)
					script := shellFunction + "\ngit-dispatch " + tc.command + "\nresult=$?\nprintf 'PARENT_TOKEN=%s\\n' \"$GH_TOKEN\"\nexit $result\n"
					cmd := exec.Command(sh, "-f", "-c", script)
					cmd.Env = append(os.Environ(), "HOME="+dir, "PATH="+dir+":"+os.Getenv("PATH"), "TEST_REMOTE="+tc.remote, "GH_TOKEN=existing-test-token", "GITHUB_TOKEN=other-test-token")
					if tc.name == "Git failure" {
						cmd.Env = append(cmd.Env, "TEST_GIT_EXIT=23")
					}
					output, err := cmd.CombinedOutput()
					out := string(output)
					if (err != nil) != tc.fail {
						t.Fatalf("error=%v, want failure=%v\n%s", err, tc.fail, out)
					}
					if tc.name == "Git failure" && err.(*exec.ExitError).ExitCode() != 23 {
						t.Fatalf("Git exit status lost: %v", err)
					}
					if !strings.Contains(out, tc.want) {
						t.Fatalf("missing %q\n%s", tc.want, out)
					}
					if strings.Contains(out, "<selected-test-token>") {
						t.Fatal("token exposed in Git arguments")
					}
					if !strings.Contains(out, "PARENT_TOKEN=existing-test-token") {
						t.Fatalf("parent credentials changed: %s", out)
					}
					if tc.name == "missing account" && strings.Contains(out, "TOKEN=existing-test-token\nSSH=") {
						t.Fatalf("Git ran after auth failed: %s", out)
					}
					calls, _ := os.ReadFile(filepath.Join(dir, "gh-calls"))
					if tc.name == "SSH key" && len(calls) != 0 {
						t.Fatalf("SSH invoked gh: %s", calls)
					}
					if strings.Contains(string(calls), "auth switch") {
						t.Fatal("global account switched")
					}
					if strings.Contains(tc.want, "selected-test-token") && !strings.Contains(out, "credential.helper=") {
						t.Fatalf("credential helpers not overridden: %s", out)
					}
				})
			}
		})
	}
}

func TestStandaloneDispatcherMatchesCanonical(t *testing.T) {
	data, err := os.ReadFile("../../add-to-bashrc-zshrc.sh")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(data)) != strings.TrimSpace(shellFunction) {
		t.Fatal("standalone dispatcher differs from canonical shellFunction")
	}
}
