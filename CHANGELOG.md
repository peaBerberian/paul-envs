# Changelog

## Unreleased

### Changes

- make `mise` opt-out, not opt-in, as it is basically needed for fine-grained versions for language tooling
- tui: separate agent step from tools step to be more readable
- for languages `latest` now means the latest published stable version of a language, not whatever is present in Ubuntu's repositories

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
