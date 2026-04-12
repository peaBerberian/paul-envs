# Changelog

## Unreleased

- Don't cache build anymore with both `podman` and `docker` so a `build` always lead to the predictible and wanted base
- `paul-env run <name> <command>` now always runs `command` in the (default) shell
- Still set base directory envs (e.g. `XDG_*`) for login shells

## v0.6.0 (2026-04-12)

- Make it compatible with `podman-compose` to enable rootless usage
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
