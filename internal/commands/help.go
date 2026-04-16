package commands

import (
	"github.com/peaberberian/paul-envs/internal/console"
	"github.com/peaberberian/paul-envs/internal/files"
)

func Help(filestore *files.FileStore, console *console.Console) {
	_ = filestore

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
  paul-envs <command> [arguments]
  paul-envs help <command>

Commands:
  create       Create a project configuration
  list         List projects
  build        Build a project image
  run          Run or join a project container
  remove       Remove one project and its managed assets
  version      Show paul-envs and container engine versions
  completion   Print shell completion scripts
  help         Show global or per-command help
  interactive  Start the guided interactive flow
  clean        Remove global paul-envs data and managed assets across projects

Run 'paul-envs <command> --help' for command-specific flags and examples.
`)
}
