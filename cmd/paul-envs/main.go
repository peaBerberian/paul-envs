package main

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"

	"github.com/peaberberian/paul-envs/internal/commands"
	"github.com/peaberberian/paul-envs/internal/console"
	"github.com/peaberberian/paul-envs/internal/files"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(),
		os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Console: Handle input/output messages
	console := console.New(ctx, os.Stdin, os.Stdout, os.Stderr)

	// FileStore: Handle Compose and Environment files
	filestore, err := files.NewFileStore()
	if err != nil {
		console.Error("Error: %v", err)
		os.Exit(1)
	}

	if len(os.Args) < 2 {
		commands.Help(filestore, console)
		os.Exit(0)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	cmdErr := runCommand(ctx, cmd, args, filestore, console)
	if errors.Is(cmdErr, errUnknownCommand) {
		console.Error("Error: unknown command: %s", cmd)
		console.Error("Run with --help to have a list of authorized commands")
		os.Exit(1)
	}

	if cmdErr != nil {
		if errors.Is(cmdErr, context.Canceled) {
			console.Error("\nOperation cancelled")
			os.Exit(130)
		}
		console.Error("Error: %v", cmdErr)
		os.Exit(1)
	}
}

var errUnknownCommand = errors.New("unknown command")

func runCommand(
	ctx context.Context,
	cmd string,
	args []string,
	filestore *files.FileStore,
	console *console.Console,
) error {
	switch cmd {
	case "create", "c", "--create", "-c":
		return commands.Create(args, filestore, console)
	case "list", "ls", "l", "--list", "-l":
		return commands.List(ctx, args, filestore, console)
	case "build", "b", "--build", "-b":
		return commands.Build(ctx, args, filestore, console)
	case "run", "e", "--run", "-e":
		return commands.Run(ctx, args, filestore, console)
	case "remove", "rm", "r", "--remove", "-r":
		return commands.Remove(ctx, args, filestore, console)
	case "version", "v", "--version", "-v":
		return commands.Version(ctx, console)
	case "clean", "x", "--clean", "-x":
		return commands.Clean(ctx, args, filestore, console)
	case "interactive", "i", "--interactive", "-i":
		return commands.Interactive(ctx, filestore, console)
	case "help", "h", "--help", "-h":
		if len(args) == 0 || isHelpCommand(args[0]) {
			commands.Help(filestore, console)
			return nil
		}
		return runCommand(ctx, args[0], append([]string{"--help"}, args[1:]...), filestore, console)
	default:
		return errUnknownCommand
	}
}

func isHelpCommand(arg string) bool {
	switch arg {
	case "help", "h", "--help", "-h":
		return true
	default:
		return false
	}
}
