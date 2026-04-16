package commands

import (
	"context"
	"errors"
	"flag"
	"strings"

	"github.com/peaberberian/paul-envs/internal/console"
	"github.com/peaberberian/paul-envs/internal/files"
)

func Interactive(ctx context.Context, args []string, fs *files.FileStore, c *console.Console) error {
	flagset := newCommandFlagSet("interactive", c)
	flagset.Usage = func() {
		writeCommandUsage(
			c,
			flagset,
			"paul-envs interactive [flags]",
			"Start the guided interactive flow.",
		)
	}
	if err := parseCommandFlags(flagset, args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	for {
		c.WriteLn("")
		c.Info("Available commands:")
		c.WriteLn("  1. create  - Create a new configuration for a project directory")
		c.WriteLn("  2. list    - List all created configurations")
		c.WriteLn("  3. build   - Build an image according to your configuration")
		c.WriteLn("  4. run     - Run a container based on a built image")
		c.WriteLn("  5. remove  - Remove a configuration and its data")
		c.WriteLn("  6. version - Show the current version")
		c.WriteLn("  7. clean   - Remove stored paul-envs data and container assets")
		c.WriteLn("  8. exit    - Exit interactive mode")
		c.WriteLn("")

		choice, err := c.AskString("Select a command (1-7 or name)", "")
		if err != nil {
			return err
		}

		choice = strings.ToLower(strings.TrimSpace(choice))

		var cmdErr error
		switch choice {
		case "1", "create":
			path, err := c.AskString("Project path", "")
			if err != nil {
				return err
			}
			cmdErr = Create([]string{path}, fs, c)
		case "2", "list", "ls":
			cmdErr = List(ctx, []string{}, fs, c)
		case "3", "build":
			cmdErr = Build(ctx, []string{}, fs, c)
		case "4", "run":
			cmdErr = Run(ctx, []string{}, fs, c)
		case "5", "remove", "rm":
			cmdErr = Remove(ctx, []string{}, fs, c)
		case "6", "version":
			cmdErr = Version(ctx, []string{}, c)
		case "7", "clean":
			cmdErr = Clean(ctx, []string{}, fs, c)
		case "8", "exit", "quit", "q":
			c.Success("Goodbye!")
			return nil
		default:
			c.Error("Invalid choice: %s", choice)
			continue
		}

		if cmdErr != nil {
			c.Error("Command failed: %v", cmdErr)
			continuePrompt, err := c.AskYesNo("Continue?", true)
			if err != nil || !continuePrompt {
				return err
			}
		}
	}
}
