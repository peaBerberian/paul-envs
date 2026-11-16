package commands

import (
	"context"
	"errors"
	"flag"
	"fmt"

	versions "github.com/peaberberian/paul-envs/internal"
	"github.com/peaberberian/paul-envs/internal/console"
	"github.com/peaberberian/paul-envs/internal/engine"
)

func Version(ctx context.Context, args []string, console *console.Console) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	flagset := newCommandFlagSet("version", console)
	short := flagset.Bool("short", false, "Print only the application version, without detecting the container engine.")
	flagset.Usage = func() {
		writeCommandUsage(
			console,
			flagset,
			"paul-envs version [flags]",
			"Show the current paul-envs version and the detected container engine version.",
		)
	}
	if err := parseCommandFlags(flagset, args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	if *short {
		console.WriteLn("%s", versions.Version.ToString())
		return nil
	}

	console.WriteLn("paul-envs version %d.%d.%d",
		versions.Version.Major, versions.Version.Minor, versions.Version.Patch)
	containerEngine, err := engine.New(ctx, console)
	if err != nil {
		return fmt.Errorf("failed to fetch information on container engine: %w", err)
	}
	info, err := containerEngine.Info(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch information on container engine: %w", err)
	}
	console.WriteLn("Container engine: %s", info.Name)
	console.WriteLn("Container engine version: %s", info.Version)
	return nil
}
