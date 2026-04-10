package shell

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const BlockStart = "### Managed by Bipolar ###"
const BlockEnd = "### Bipolar End ###"

// shellFunction is the canonical git-dispatch function body, baked into the binary.
const shellFunction = `git-dispatch() {
    local network_cmds="push pull fetch clone"
    local conf_file="$HOME/.git_profiles.conf"
    local remote_url=""

    # Only intercept network-heavy commands
    if [[ " $network_cmds " =~ " $1 " ]]; then
        # Identify the URL
        if [[ "$1" == "clone" ]]; then
            for arg in "$@"; do [[ "$arg" =~ ":" || "$arg" =~ "/" ]] && remote_url="$arg"; done
        else
            remote_url=$(command git config --get remote.origin.url 2>/dev/null)
        fi

        # DO NOT intercept if it's HTTPS (must contain @ or ssh://)
        if [[ -n "$remote_url" ]] && [[ "$remote_url" =~ "@" || "$remote_url" =~ "ssh://" ]]; then

            if [[ -f "$conf_file" ]]; then
                local org_name=$(echo "$remote_url" | sed -E 's/.*[:\/]([^\/]+)\/[^\/]+$/\1/')
                local current_profile="" key_file="" org_pattern=""

                # Parse with compatibility for Zsh/Bash
                while read -r line || [[ -n "$line" ]]; do
                    # Match [Profile]
                    if echo "$line" | grep -q "^\[.*\]$"; then
                        current_profile=$(echo "$line" | tr -d '[]')
                    # Match key_file=
                    elif echo "$line" | grep -q "^key_file="; then
                        key_file=$(echo "$line" | cut -d'=' -f2- | xargs)
                    # Match org_pattern=
                    elif echo "$line" | grep -q "^org_pattern="; then
                        org_pattern=$(echo "$line" | cut -d'=' -f2- | xargs)

                        # Match found?
                        if [[ "$org_name" =~ $org_pattern ]]; then
                            if [[ "$current_profile" != "Default" ]]; then
                                echo -e "\033[0;35m[Git-Bipolar]\033[0m Org: \033[1m$org_name\033[0m | Using \033[0;36m$current_profile\033[0m Profile..."
                            fi

                            eval local expanded_key_path="$key_file"
                            if [[ -f "$expanded_key_path" ]]; then
                                GIT_SSH_COMMAND="ssh -i $expanded_key_path -o IdentitiesOnly=yes" command git "$@"
                                return $?
                            else
                                echo -e "\033[0;33m[Git-Bipolar]\033[0m Warning: key file not found: $expanded_key_path — falling back to system default"
                            fi
                        fi
                    fi
                done < "$conf_file"
            fi
        fi
    fi

    # Fallback for non-network, HTTPS, or unmatched profiles
    command git "$@"
}

alias git='git-dispatch'`

// ManagedBlock is the full block as it would appear in the rc file.
var ManagedBlock = BlockStart + "\n" + shellFunction + "\n" + BlockEnd

// DetectShell returns the shell base name and the path to its rc file.
func DetectShell() (string, string, error) {
	shellPath := os.Getenv("SHELL")
	if shellPath == "" {
		return "", "", fmt.Errorf("SHELL environment variable not set")
	}

	shellName := filepath.Base(shellPath)
	home, err := os.UserHomeDir()
	if err != nil {
		return "", "", fmt.Errorf("cannot determine home directory: %w", err)
	}

	var rcFile string
	switch shellName {
	case "zsh":
		rcFile = filepath.Join(home, ".zshrc")
	case "bash":
		bashrc := filepath.Join(home, ".bashrc")
		bashProfile := filepath.Join(home, ".bash_profile")
		if _, err := os.Stat(bashrc); err == nil {
			rcFile = bashrc
		} else if _, err := os.Stat(bashProfile); err == nil {
			rcFile = bashProfile
		} else {
			rcFile = bashrc // will be created if needed
		}
	default:
		rcFile = filepath.Join(home, ".profile")
	}

	return shellName, rcFile, nil
}

// IsInstalledInRCFile returns true if the managed block marker is present in the rc file.
func IsInstalledInRCFile(rcFile string) bool {
	data, err := os.ReadFile(rcFile)
	if err != nil {
		return false
	}
	return strings.Contains(string(data), BlockStart)
}

// CheckRCFile returns whether the block exists and whether it matches the canonical version.
func CheckRCFile(rcFile string) (exists bool, upToDate bool, err error) {
	data, err := os.ReadFile(rcFile)
	if os.IsNotExist(err) {
		return false, false, nil
	}
	if err != nil {
		return false, false, fmt.Errorf("cannot read %s: %w", rcFile, err)
	}

	text := string(data)
	startIdx := strings.Index(text, BlockStart)
	endIdx := strings.Index(text, BlockEnd)

	if startIdx < 0 || endIdx < 0 {
		return false, false, nil
	}

	endIdx += len(BlockEnd)
	existing := strings.TrimSpace(text[startIdx:endIdx])
	canonical := strings.TrimSpace(ManagedBlock)

	return true, existing == canonical, nil
}

// InstallToRCFile appends the managed block to the rc file.
func InstallToRCFile(rcFile string) error {
	existing := ""
	data, err := os.ReadFile(rcFile)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("cannot read %s: %w", rcFile, err)
	}
	if err == nil {
		existing = string(data)
	}

	newContent := existing
	if newContent != "" && !strings.HasSuffix(newContent, "\n") {
		newContent += "\n"
	}
	newContent += "\n" + ManagedBlock + "\n"

	return os.WriteFile(rcFile, []byte(newContent), 0644)
}

// UpdateInRCFile replaces the managed block in the rc file with the current canonical version.
func UpdateInRCFile(rcFile string) error {
	data, err := os.ReadFile(rcFile)
	if err != nil {
		return fmt.Errorf("cannot read %s: %w", rcFile, err)
	}

	text := string(data)
	startIdx := strings.Index(text, BlockStart)
	endIdx := strings.Index(text, BlockEnd)

	if startIdx < 0 || endIdx < 0 {
		return InstallToRCFile(rcFile)
	}
	endIdx += len(BlockEnd)

	newText := text[:startIdx] + ManagedBlock + text[endIdx:]
	return os.WriteFile(rcFile, []byte(newText), 0644)
}

// ProfilesConfPath returns the path to the git profiles config file.
func ProfilesConfPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".git_profiles.conf")
}

// CheckProfilesConf returns true if the profiles config file exists.
func CheckProfilesConf() bool {
	_, err := os.Stat(ProfilesConfPath())
	return err == nil
}

// CreateDefaultProfilesConf creates an empty profiles config.
func CreateDefaultProfilesConf() error {
	return os.WriteFile(ProfilesConfPath(), []byte(""), 0600)
}

// EnsureInstalled checks for the rc function block and profiles config,
// silently installing any missing pieces on first run.
func EnsureInstalled() error {
	shellName, rcFile, err := DetectShell()
	if err != nil {
		return err
	}

	installed := IsInstalledInRCFile(rcFile)
	profExists := CheckProfilesConf()

	if installed && profExists {
		return nil
	}

	fmt.Printf("\n\033[1;33m[Bipolar] First-time setup\033[0m\n")
	fmt.Printf("Shell: %s  |  RC file: %s\n\n", shellName, rcFile)

	if !installed {
		fmt.Printf("  Installing git-dispatch function into %s... ", rcFile)
		if err := InstallToRCFile(rcFile); err != nil {
			fmt.Printf("\033[31mfailed: %v\033[0m\n", err)
		} else {
			fmt.Printf("\033[32m✓\033[0m\n")
		}
	}

	if !profExists {
		fmt.Printf("  Creating default profiles config at %s... ", ProfilesConfPath())
		if err := CreateDefaultProfilesConf(); err != nil {
			fmt.Printf("\033[31mfailed: %v\033[0m\n", err)
		} else {
			fmt.Printf("\033[32m✓\033[0m\n")
		}
	}

	if !installed {
		fmt.Printf("\nRun \033[1msource %s\033[0m to activate in your current session.\n\n", rcFile)
	}

	return nil
}
