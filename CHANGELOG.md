# Changelog

## Unreleased

### Changes

- `dotfiles` is now in a project-specific directory alongside `build.conf` and `run.conf`
- `GIT_AUTHOR_NAME` and `GIT_AUTHOR_EMAIL` are now part of `run.conf`, and do not necessitate a re-build when updated
- `create` command flags now don't silently disable other unspecified args (e.g. `--neovim` which - as a tool - would imply no other tool) in CLI mode, confirmation is asked instead

### Features

- dotfiles now only apply at `run`-time. Updating dotfiles does not lead to a re-build
- Add `DOTFILES_PATH` to `run.conf` to let user configure where dotfiles are
- base "dotfiles" can be seeded from e.g. `XDG_CONFIG_HOME/paul-envs/dotfiles`
- `help` flag per-command

### Bug fixes

- Restore explicit language version configuration in the TUI

## v0.7.0 (2026-04-14)

### Changes

- Remove `.env` and `compose.yaml` configuration in profit of `build.conf` and `run.conf` files whose influence (necessitate a re-build or not) is much clearer
- make `mise` opt-out, not opt-in, as it is basically needed for fine-grained versions for language tooling
- for languages `latest` now means the latest published stable version of a language, not whatever is present in Ubuntu's repositories
- In the project config directory, put the internal `paul-envs` stuff in `.paul-envs` directory so it's given less importance than the actual config

### Features

- Stop the need for a `compose`-compatible tool (`docker compose` or `podman-compose`), just the base container engine (`docker` or `podman`) is needed now
- Don't re-build if only run-associated config is fixed
- tui: separate agent step from tools step to be more readable

### Bug fixes

- `run` commands (e.g. `paul-envs run my-project <COMMANDS>`) now can be anything availabe when running the shell, including mise-installed tools
- Ensure `mise` make language available when they are configured to be `latest`
- Still set base directory envs (e.g. `XDG_*`) for login shells

## v0.6.0 (2026-04-12)

### Features

- Make it compatible with `podman-compose` to enable rootless usage

### Bug fixes

- Store `.claude` and `.codex` in `$XDG_STATE_HOME` so they persist

## v0.5.0 (2026-02-08)

### Features

- Add `firefox` web browser
- Add `opencode` LLM "agentic" tool

## v0.4.0 (2026-02-06)

### Features

- Add `claude-Code` LLM "agentic" tool
- Add `codex` LLM "agentic" tool

## v0.3.0 (2025-12-10)

### Features

- Add `delta` pager

### Bug fixes

- Replace with newer Dockerfile on `build` and `create`

## v0.2.0 (2025-12-09)

### Features

- Add `Oh My Posh` prompt (due to better support of jujutsu)

### Bug fixes

- `version`: fix `version` command formatting for the tool's version

## v0.1.0 (2025-12-06)

Initial versioned release after rewrite in Go (from BASH)
