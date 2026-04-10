git-dispatch() {
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
        
        # 2. DO NOT intercept if it's HTTPS (must contain @ or ssh://)
        if [[ -n "$remote_url" ]] && [[ "$remote_url" =~ "@" || "$remote_url" =~ "ssh://" ]]; then
            
            if [[ -f "$conf_file" ]]; then
                local org_name=$(echo "$remote_url" | sed -E 's/.*[:\/]([^\/]+)\/[^\/]+$/\1/')
                local current_profile="" key_file="" org_pattern=""

                # 3. Parse with compatibility for Zsh/Bash
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
                            # 1. If it's the "Default" profile, print nothing
                            if [[ "$current_profile" != "Default" ]]; then
                                echo -e "\033[0;35m[Git-Identity]\033[0m Org: \033[1m$org_name\033[0m | Using \033[0;36m$current_profile\033[0m Profile..."
                            fi

                            eval local expanded_key_path="$key_file"
                            if [[ -f "$expanded_key_path" ]]; then
                                GIT_SSH_COMMAND="ssh -i $expanded_key_path -o IdentitiesOnly=yes" command git "$@"
                                return $?
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

alias git='git-dispatch'
