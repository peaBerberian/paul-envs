package commands

import (
	"github.com/peaberberian/paul-envs/internal/console"
	"github.com/peaberberian/paul-envs/internal/files"
)

func Help(filestore *files.FileStore, console *console.Console) {
	// TODO: only list commands and have `--help` flags per command
	console.WriteLn(`paul-envs - Development Environment Manager

This tool simplify the configuration of per-project container images by providing a base
with sane defaults for CLI-oriented development workflows (neovim, atuin etc.).

The workflow is:
1. You "create" a configuration
2. You "build" an image corresponding to that config.
   You can edit the configuration at any time and then re-build the image.
	 You can also just re-build to update the tools in the image.
3. You "run" that image, also mounting your project in it.

Usage:
  paul-envs create <path> [options]
  paul-envs list
  paul-envs build <name>
  paul-envs run <name> [commands]
  paul-envs remove <name>
  paul-envs version
  paul-envs help
  paul-envs interactive
  paul-envs clean

Options for create (all optional):
  --no-prompt              Non-interactive mode (uses defaults)
  --seed-dotfiles          Seed the project dotfiles/ directory from
                           $XDG_CONFIG_HOME/paul-envs/dotfiles
                           In interactive mode, paul-envs will ask about this
                           automatically if that global template exists and is
                           non-empty
  --name NAME              Name of this project (default: directory name)
  --uid UID                Container UID (default: current user - or 1000 on windows)
  --gid GID                Container GID (default: current group - or 1000 on windows)
  --username NAME          Container username (default: dev)
  --shell SHELL            User shell: bash|zsh|fish (prompted if not specified)
  --nodejs VERSION         Node.js installation:
                             'none' - skip installation of Node.js
                             'latest' - use latest version
                             '20.10.0' - specific version
                           (prompted if no language specified)
  --rust VERSION           Rust installation:
                             'none' - skip installation of Rust
                             'latest' - use latest version
                             '1.75.0' - specific version
                           (prompted if no language specified)
  --python VERSION         Python installation:
                             'none' - skip installation of Python
                             'latest' - use latest version
                             '3.12.0' - specific version
                           (prompted if no language specified)
  --go VERSION             Go installation:
                             'none' - skip installation of Go
                             'latest' - use latest version
                             '1.21.5' - specific version
                           (prompted if no language specified)
  --enable-wasm            Add WASM-specialized tools (binaryen, Rust wasm target if enabled)
                           (prompted if no language specified)
  --enable-ssh             Enable ssh access on port 22 (E.g. to access files from your host)
                           (prompted if not specified)
  --enable-sudo            Enable sudo access in container with a "dev" password
                           (prompted if not specified)
  --git-name NAME          Git user.name (optional)
  --git-email EMAIL        Git user.email (optional)
  --neovim                 Install Neovim (text editor)
                           (prompted if no tool specified)
  --starship               Install Starship (prompt)
                           (prompted if no tool specified)
  --oh-my-posh             Install Oh My Posh (prompt)
                           (prompted if no tool specified)
  --atuin                  Install Atuin (shell history)
                           (prompted if no tool specified)
                           (prompted if no tool specified)
  --zellij                 Install Zellij (terminal multiplexer)
                           (prompted if no tool specified)
  --jujutsu                Install Jujutsu (Git-compatible VCS)
                           (prompted if no tool specified)
  --delta                  Install Delta (Colored pager for e.g. Git)
                           (prompted if no tool specified)
  --open-code              Install opencode (LLM tool)
                           (prompted if no tool specified)
  --claude-code            Install Claude Code (LLM tool)
                           (prompted if no tool specified)
  --codex                  Install OpenAI's codex (LLM tool)
                           (prompted if no tool specified)
  --firefox                Install Mozilla Firefox (web browser)
                           (prompted if no tool specified)
  --no-mise                Prevent the installation of "mise", which is used to install
                         	 languages and related tool (e.g. "node", "rust"/"cargo" etc.).
													 When disabled, we will only rely on Ubuntu's repositories for
													 language tools installation which may be old (yet stable) versions.
  --package PKG_NAME       Additional Ubuntu package (prompted if not specified, can be repeated)
  --port PORT              Expose container port (prompted if not specified, can be repeated)
  --volume HOST:CONT[:ro]  Mount volume (prompted if not specified, can be repeated)

Windows/Git Bash Notes:
  - UID/GID default to 1000 on Windows (Docker Desktop requirement)

Creating a configuration in interactive Mode (default):
  paul-envs create ~/projects/myapp
  # Will prompt for all unspecified options

Creating a configuration in a non-Interactive Mode:
  paul-envs create ~/projects/myapp --no-prompt --shell bash --nodejs latest

Creating a configuration from a global dotfiles template:
  paul-envs create ~/projects/myapp --no-prompt --seed-dotfiles
  # Copies from $XDG_CONFIG_HOME/paul-envs/dotfiles into the new project's
  # dotfiles/ directory

Mixed Mode (some flags + prompts):
  paul-envs create ~/projects/myapp --nodejs 20.10.0 --rust latest --no-mise
  # Will prompt for shell, sudo, packages, ports, and volumes

Full Configuration Example:
  paul-envs create ~/work/api \
    --name myApp \
    --shell zsh \
    --nodejs 20.10.0 \
    --rust latest \
    --python 3.12.0 \
    --go latest \
    --neovim \
    --starship \
    --zellij \
    --jujutsu \
    --delta \
    --open-code \
    --claude-code \
    --codex \
    --firefox \
    --enable-ssh \
    --enable-sudo \
    --git-name "John Doe" \
    --git-email "john@example.com" \
    --package ripgrep \
    --package ripgrep \
    --port 3000 \
    --port 5432 \
    --volume ~/.git-credentials:/home/dev/.git-credentials:ro

NOTE: To start a guided prompt, you can also just run:
  paul-envs interactive
`)
}
