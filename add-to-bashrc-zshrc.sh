git-dispatch() {
    local network_cmds="push pull fetch clone ls-remote"
    local conf_file="$HOME/.git_profiles.conf"
    local remote_url=""

    # Identify the URL
    if [[ "$1" == "clone" ]]; then
        for arg in "$@"; do
            if [[ "$arg" =~ ^https?:// || "$arg" =~ ^ssh:// || "$arg" =~ ^[^/]+@[^:]+: ]]; then
                remote_url="$arg"
                break
            fi
        done
    else
        remote_url=$(command git config --get remote.origin.url 2>/dev/null)
    fi

    if [[ -n "$remote_url" ]] && [[ -f "$conf_file" ]]; then
        local org_name=$(echo "$remote_url" | sed -E 's/.*[:\/]([^\/]+)\/[^\/]+$/\1/')
        local current_profile="" key_file="" git_name="" git_email="" gh_user="" sign="" sign_key="" org_pattern=""

        # Parse with compatibility for Zsh/Bash
        while read -r line || [[ -n "$line" ]]; do
            if echo "$line" | grep -q "^\[.*\]$"; then
                current_profile=$(echo "$line" | tr -d '[]')
                key_file="" git_name="" git_email="" gh_user="" sign="" sign_key="" org_pattern=""
            elif echo "$line" | grep -q "^key_file="; then
                key_file=$(echo "$line" | cut -d'=' -f2- | xargs)
            elif echo "$line" | grep -q "^git_name="; then
                git_name=$(echo "$line" | cut -d'=' -f2-)
            elif echo "$line" | grep -q "^git_email="; then
                git_email=$(echo "$line" | cut -d'=' -f2-)
            elif echo "$line" | grep -q "^gh_user="; then
                gh_user=$(echo "$line" | cut -d'=' -f2- | xargs)
            elif echo "$line" | grep -q "^sign="; then
                sign=$(echo "$line" | cut -d'=' -f2- | xargs)
            elif echo "$line" | grep -q "^sign_key="; then
                sign_key=$(echo "$line" | cut -d'=' -f2- | xargs)
            elif echo "$line" | grep -q "^org_pattern="; then
                org_pattern=$(echo "$line" | cut -d'=' -f2- | xargs)

                if [[ "$org_name" =~ $org_pattern ]]; then
                    if [[ "$current_profile" != "Default" ]]; then
                        local sign_note=""
                        [[ "$sign" == "true" && -n "$sign_key" ]] && sign_note="  \033[0;32m🔏 signing on\033[0m"
                        echo -e "\033[0;35m[Git-Bipolar]\033[0m Org: \033[1m$org_name\033[0m | Using \033[0;36m$current_profile\033[0m Profile...${sign_note}"
                    fi

                    # Build identity override args (-c is git's highest-precedence layer,
                    # overriding both local .git/config and global ~/.gitconfig)
                    local id_args=()
                    [[ -n "$git_name" ]] && id_args+=(-c "user.name=$git_name")
                    [[ -n "$git_email" ]] && id_args+=(-c "user.email=$git_email")

                    # Commit signing (SSH): inject signing config and show an indicator
                    if [[ "$sign" == "true" && -n "$sign_key" ]]; then
                        eval local expanded_sign_key="$sign_key"
                        id_args+=(-c "gpg.format=ssh" -c "user.signingkey=$expanded_sign_key" -c "commit.gpgsign=true" -c "tag.gpgsign=true")
                        case "$1" in
                            commit|tag|merge|rebase|cherry-pick|revert|am|commit-tree)
                                echo -e "\033[0;35m[Git-Bipolar]\033[0m \033[0;32m🔏 Signing commit with\033[0m \033[1m$sign_key\033[0m" ;;
                        esac
                    fi

                    # GitHub HTTPS auth is separate from commit identity. Resolve the
                    # configured account without changing gh's globally active user.
                    if [[ " $network_cmds " =~ " $1 " ]] && [[ "$remote_url" =~ ^https://github[.]com/ ]] && [[ -n "$gh_user" ]]; then
                        local gh_token
                        if ! gh_token=$(GH_TOKEN= GITHUB_TOKEN= command gh auth token --hostname github.com --user "$gh_user") || [[ -z "$gh_token" ]]; then
                            printf '[Git-Bipolar] Cannot load GitHub credentials for %s. Run gh auth login for that account.\n' "$gh_user" >&2
                            return 1
                        fi
                        # Reset other helpers so cached credentials cannot win. The
                        # token stays in the child environment, never Git arguments.
                        GH_TOKEN="$gh_token" GITHUB_TOKEN= command git "${id_args[@]}" \
                            -c credential.helper= -c 'credential.https://github.com.helper=!gh auth git-credential' "$@"
                        return $?
                    fi

                    # SSH network commands: inject key + identity
                    if [[ " $network_cmds " =~ " $1 " ]] && [[ "$remote_url" =~ "@" || "$remote_url" =~ "ssh://" ]]; then
                        eval local expanded_key_path="$key_file"
                        if [[ -f "$expanded_key_path" ]]; then
                            GIT_SSH_COMMAND="ssh -i $expanded_key_path -o IdentitiesOnly=yes" command git "${id_args[@]}" "$@"
                            return $?
                        else
                            echo -e "\033[0;33m[Git-Bipolar]\033[0m Warning: key file not found: $expanded_key_path — falling back to system default"
                        fi
                    fi

                    # All other commands: inject identity if configured
                    if [[ ${#id_args[@]} -gt 0 ]]; then
                        command git "${id_args[@]}" "$@"
                        return $?
                    fi

                    command git "$@"
                    return $?
                fi
            fi
        done < "$conf_file"
    fi

    # Fallback for unmatched profiles or no remote
    command git "$@"
}

alias git='git-dispatch'
