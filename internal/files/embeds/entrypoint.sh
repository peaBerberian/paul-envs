#!/bin/bash

set -eu

# This is the container's entry point. All containers start here so runtime
# state, shell overrides, and optional services can be initialized before the
# user shell or command runs.

CONTAINER_USERNAME="${CONTAINER_USERNAME:-dev}"
USER_SHELL="${USER_SHELL:-/usr/bin/bash}"
HOME_DIR="/home/${CONTAINER_USERNAME}"
INITIAL_CACHE_DIR="${INITIAL_CACHE_DIR:-${HOME_DIR}/.initial-cache}"
INITIAL_LOCAL_DIR="${INITIAL_LOCAL_DIR:-${HOME_DIR}/.initial-local}"
CONTAINER_CACHE_DIR="${CONTAINER_CACHE_DIR:-${HOME_DIR}/.container-cache}"
CONTAINER_LOCAL_DIR="${CONTAINER_LOCAL_DIR:-${HOME_DIR}/.container-local}"
CACHE_MARKER="${CONTAINER_CACHE_DIR}/.initialized"
LOCAL_MARKER="${CONTAINER_LOCAL_DIR}/.initialized"
DOTFILES_MOUNT_DIR="${DOTFILES_MOUNT_DIR:-/paul-env/dotfiles}"

ensure_managed_block() {
    target_file="$1"
    shell_kind="$2"

    mkdir -p "$(dirname "$target_file")"
    touch "$target_file"

    if grep -qF "paul-envs managed ${shell_kind} overrides" "$target_file"; then
        return
    fi

    case "$shell_kind" in
        bash)
            cat >> "$target_file" <<'EOF'
# paul-envs managed bash overrides
[ -f ~/.container-overrides.bash ] && source ~/.container-overrides.bash
EOF
            ;;
        zsh)
            cat >> "$target_file" <<'EOF'
# paul-envs managed zsh overrides
[ -f ~/.container-overrides.zsh ] && source ~/.container-overrides.zsh
EOF
            ;;
        fish)
            cat >> "$target_file" <<'EOF'
# paul-envs managed fish overrides
if test -f ~/.container-overrides.fish
    source ~/.container-overrides.fish
end
EOF
            ;;
    esac
}

write_shell_overrides() {
    cat > "${HOME_DIR}/.container-overrides.bash" <<EOF
export XDG_CACHE_HOME="${HOME_DIR}/.container-cache/cache"
export XDG_STATE_HOME="${HOME_DIR}/.container-local/state"
export XDG_DATA_HOME="${HOME_DIR}/.container-local/data"
export _ZO_DATA_DIR="${HOME_DIR}/.container-local/zoxide"
export STARSHIP_CACHE="${HOME_DIR}/.container-local/starship"
export ATUIN_DB_PATH="${HOME_DIR}/.container-local/atuin/history.db"
export HISTFILE="${HOME_DIR}/.container-local/.bash_history"
export PIP_CACHE_DIR="\$HOME/.container-cache/pip"
export GOPATH="\$HOME/.container-local/gopath"
export GOMODCACHE="\$HOME/.container-cache/go/mod"
export PATH="\$HOME/.local/bin:\$HOME/.opencode/bin:\$GOPATH/bin:\$PATH"
if [ -f "\$HOME/.cargo/env" ]; then
    . "\$HOME/.cargo/env"
fi
if command -v mise >/dev/null 2>&1; then
    eval "\$(mise activate bash)"
fi
if command -v starship >/dev/null 2>&1; then
    eval "\$(starship init bash)"
fi
if command -v oh-my-posh >/dev/null 2>&1; then
    eval "\$(oh-my-posh init bash)"
fi
if command -v atuin >/dev/null 2>&1; then
    eval "\$(atuin init bash)"
fi
EOF

    cat > "${HOME_DIR}/.container-overrides.zsh" <<EOF
export XDG_CACHE_HOME="${HOME_DIR}/.container-cache/cache"
export XDG_STATE_HOME="${HOME_DIR}/.container-local/state"
export XDG_DATA_HOME="${HOME_DIR}/.container-local/data"
export _ZO_DATA_DIR="${HOME_DIR}/.container-local/zoxide"
export STARSHIP_CACHE="${HOME_DIR}/.container-local/starship"
export ATUIN_DB_PATH="${HOME_DIR}/.container-local/atuin/history.db"
export HISTFILE="${HOME_DIR}/.container-local/.zsh_history"
export PIP_CACHE_DIR="\$HOME/.container-cache/pip"
export GOPATH="\$HOME/.container-local/gopath"
export GOMODCACHE="\$HOME/.container-cache/go/mod"
export PATH="\$HOME/.local/bin:\$HOME/.opencode/bin:\$GOPATH/bin:\$PATH"
if [ -f "\$HOME/.cargo/env" ]; then
    . "\$HOME/.cargo/env"
fi
if command -v mise >/dev/null 2>&1; then
    eval "\$(mise activate zsh)"
fi
if command -v starship >/dev/null 2>&1; then
    eval "\$(starship init zsh)"
fi
if command -v oh-my-posh >/dev/null 2>&1; then
    eval "\$(oh-my-posh init zsh)"
fi
if command -v atuin >/dev/null 2>&1; then
    eval "\$(atuin init zsh)"
fi
EOF

    cat > "${HOME_DIR}/.container-overrides.fish" <<EOF
set -gx XDG_CACHE_HOME ${HOME_DIR}/.container-cache/cache
set -gx XDG_STATE_HOME ${HOME_DIR}/.container-local/state
set -gx XDG_DATA_HOME ${HOME_DIR}/.container-local/data
set -gx _ZO_DATA_DIR ${HOME_DIR}/.container-local/zoxide
set -gx STARSHIP_CACHE ${HOME_DIR}/.container-local/starship
set -gx ATUIN_DB_PATH ${HOME_DIR}/.container-local/atuin/history.db
set -gx PIP_CACHE_DIR \$HOME/.container-cache/pip
set -gx GOPATH \$HOME/.container-local/gopath
set -gx GOMODCACHE \$HOME/.container-cache/go/mod
set -gx PATH \$HOME/.local/bin \$HOME/.opencode/bin \$GOPATH/bin \$PATH
if test -f \$HOME/.cargo/env
    set -gx PATH \$HOME/.cargo/bin \$PATH
end
if type -q mise
    mise activate fish | source
end
if type -q starship
    starship init fish | source
end
if type -q oh-my-posh
    oh-my-posh init fish | source
end
if type -q atuin
    atuin init fish | source
end
EOF

    chown "${CONTAINER_USERNAME}:${CONTAINER_USERNAME}" \
        "${HOME_DIR}/.container-overrides.bash" \
        "${HOME_DIR}/.container-overrides.zsh" \
        "${HOME_DIR}/.container-overrides.fish"
}

sync_dotfiles() {
    if [ ! -d "$DOTFILES_MOUNT_DIR" ]; then
        return
    fi
    if [ -z "$(find "$DOTFILES_MOUNT_DIR" -mindepth 1 -print -quit 2>/dev/null)" ]; then
        return
    fi

    su "${CONTAINER_USERNAME}" -s /bin/sh -c '
        set -eu
        cd "'"$DOTFILES_MOUNT_DIR"'"
        tar \
            --exclude=.container-cache \
            --exclude=.container-local \
            --exclude=.initial-cache \
            --exclude=.initial-local \
            --exclude=.container-overrides.bash \
            --exclude=.container-overrides.zsh \
            --exclude=.container-overrides.fish \
            --exclude=.paul-env \
            --exclude=.paul-envs \
            -cf - . | tar -C "$HOME" -xf -
    '
}

apply_git_config() {
    su "${CONTAINER_USERNAME}" -s /bin/sh -c '
        set -eu
        git config --global merge.conflictstyle zdiff3
        if command -v delta >/dev/null 2>&1; then
            git config --global core.pager delta
            git config --global interactive.diffFilter "delta --color-only"
            git config --global delta.navigate true
            git config --global merge.conflictstyle diff3
            git config --global diff.colorMoved default
        fi
        if [ -n "${GIT_AUTHOR_NAME:-}" ]; then
            git config --global user.name "${GIT_AUTHOR_NAME}"
            if command -v jj >/dev/null 2>&1; then
                jj config set --user user.name "${GIT_AUTHOR_NAME}"
            fi
        fi
        if [ -n "${GIT_AUTHOR_EMAIL:-}" ]; then
            git config --global user.email "${GIT_AUTHOR_EMAIL}"
            if command -v jj >/dev/null 2>&1; then
                jj config set --user user.email "${GIT_AUTHOR_EMAIL}"
            fi
        fi
    '
}

# Initialize shared cache (only if not already initialized by another container)
if [ ! -f "$CACHE_MARKER" ]; then
    echo "Initializing shared cache..."
    mkdir -p "$CONTAINER_CACHE_DIR"
    cp -a "$INITIAL_CACHE_DIR/." "$CONTAINER_CACHE_DIR/" 2>/dev/null || true
    touch "$CACHE_MARKER"
fi
chown -R "${CONTAINER_USERNAME}:${CONTAINER_USERNAME}" "$CONTAINER_CACHE_DIR" 2>/dev/null || true

# Initialize local state (per-project, always check)
if [ ! -f "$LOCAL_MARKER" ]; then
    echo "Initializing local state..."
    mkdir -p "$CONTAINER_LOCAL_DIR"
    cp -a "$INITIAL_LOCAL_DIR/." "$CONTAINER_LOCAL_DIR/" 2>/dev/null || true
    touch "$LOCAL_MARKER"
fi
chown -R "${CONTAINER_USERNAME}:${CONTAINER_USERNAME}" "$CONTAINER_LOCAL_DIR" 2>/dev/null || true

sync_dotfiles
write_shell_overrides
ensure_managed_block "${HOME_DIR}/.bashrc" "bash"
ensure_managed_block "${HOME_DIR}/.bash_profile" "bash"
ensure_managed_block "${HOME_DIR}/.zshrc" "zsh"
ensure_managed_block "${HOME_DIR}/.zprofile" "zsh"
ensure_managed_block "${HOME_DIR}/.config/fish/config.fish" "fish"
chown "${CONTAINER_USERNAME}:${CONTAINER_USERNAME}" \
    "${HOME_DIR}/.bashrc" \
    "${HOME_DIR}/.bash_profile" \
    "${HOME_DIR}/.zshrc" \
    "${HOME_DIR}/.zprofile" \
    "${HOME_DIR}/.config/fish/config.fish"
apply_git_config

# SSH daemon setup
if [[ -d /var/run/sshd ]] && ! pgrep -x sshd >/dev/null; then
    /usr/sbin/sshd -D &
    if [[ -t 0 ]] && [[ $# -eq 0 ]]; then
        IP=$(hostname -I | awk "{print \$1}")
        echo "NOTE: Listening for ssh connections at ${CONTAINER_USERNAME}@${IP}:22"
    fi
fi

# Execute command or start shell
if [[ $# -eq 0 ]]; then
    exec su "${CONTAINER_USERNAME}" -s "${USER_SHELL}"
else
    case "$USER_SHELL" in
        *fish)
            exec su "${CONTAINER_USERNAME}" -s "${USER_SHELL}" -c 'source $HOME/.container-overrides.fish; exec $argv[1] $argv[2..]' -- "$@"
            ;;
        *zsh)
            exec su "${CONTAINER_USERNAME}" -s "${USER_SHELL}" -c 'source $HOME/.container-overrides.zsh; exec "$0" "$@"' -- "$@"
            ;;
        *)
            exec su "${CONTAINER_USERNAME}" -s "${USER_SHELL}" -c 'source $HOME/.container-overrides.bash; exec "$0" "$@"' -- "$@"
            ;;
    esac
fi
