# AGENTS.md

This repository is a Go CLI for managing per-project development containers with Docker or Podman.

## Where To Start

- CLI entrypoint: [cmd/paul-envs/main.go](/home/oscar/prog/repos/paul-envs/cmd/paul-envs/main.go)
- Command handlers: `internal/commands`
- `create` parsing/prompting: `internal/args`
- Config parsing and validation: `internal/config`
- Engine abstraction and implementations: `internal/engine`
- Filesystem state and embedded assets: `internal/files`

## Important Structure

- `main.go` only sets up context, console, and `FileStore`, then dispatches commands.
- Command-local flags use the standard library `flag` package.
- Shared command flag helpers live in [internal/commands/flags.go](/home/oscar/prog/repos/paul-envs/internal/commands/flags.go).
- `create` is special: its flag parsing and prompt flow live in `internal/args`, not the generic command helpers.

## Invariants

- Keep the split between `build.conf` and `run.conf`.
- Build-time concerns belong to `build.conf`; run-time concerns belong to `run.conf`.
- `create` generates configuration; it does not build images.
- `run.conf` changes should not force rebuilds unless intentionally promoted to build-time behavior.
- `dotfiles` are applied at run-time, not build-time.
- Non-interactive flags must avoid hidden prompts.
- Command help should work with `--help` and `-h`.
- When both Podman and Docker are available, current behavior prefers Podman and warns.

## Engine Rule

When changing image build or container lifecycle behavior, keep Docker and Podman implementations aligned unless the difference is explicitly engine-specific.

Typical files:

- [internal/engine/engine.go](/home/oscar/prog/repos/paul-envs/internal/engine/engine.go)
- [internal/engine/docker.go](/home/oscar/prog/repos/paul-envs/internal/engine/docker.go)
- [internal/engine/podman.go](/home/oscar/prog/repos/paul-envs/internal/engine/podman.go)

If a command flag affects engine behavior, thread it through typed options in `internal/engine/engine.go` and cover both engines.

## Common Edit Paths

- New command: update `cmd/paul-envs/main.go`, `internal/commands`, help output, completions, and docs if user-visible.
- New command flag: usually update the relevant file in `internal/commands`, then completions and docs.
- New `create` flag: usually update `internal/args/args.go`, related tests, and possibly config/files generation.
- Build behavior change: update command layer plus both engine implementations.
- File layout or generated asset change: inspect `internal/files` and `internal/files/embeds/*`.

## Validation

Use:

```sh
GOCACHE=/tmp/go-build go test ./...
```

The explicit `GOCACHE` is useful in restricted environments where the default Go build cache path may be read-only.

Before finishing a change, usually verify:

1. Flags still parse and `--help` output is sensible.
2. Docker and Podman were both updated if engine behavior changed.
3. The build/run config split is still respected.
4. Non-interactive mode is actually prompt-free.
5. Shell completions were updated if commands or flags changed.
