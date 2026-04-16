package commands

import (
	"context"
	"errors"
	"flag"
	"fmt"

	"github.com/peaberberian/paul-envs/internal/console"
	"github.com/peaberberian/paul-envs/internal/files"
)

func Completion(ctx context.Context, args []string, console *console.Console) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	flagset := newCommandFlagSet("completion", console)
	flagset.Usage = func() {
		writeCommandUsage(
			console,
			flagset,
			"paul-envs completion <bash|zsh|fish> [flags]",
			"Print the shell completion script for the selected shell to standard output.",
		)
	}
	if err := parseCommandFlags(flagset, args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	args = flagset.Args()
	if len(args) != 1 {
		return fmt.Errorf("expected exactly one shell argument: bash, zsh, or fish")
	}

	script, err := files.GetCompletionScript(args[0])
	if err != nil {
		return err
	}
	_, err = fmt.Fprint(console.Writer(), script)
	return err
}
