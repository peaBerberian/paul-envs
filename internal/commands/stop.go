package commands

import (
	"context"
	"errors"
	"flag"
	"fmt"

	"github.com/peaberberian/paul-envs/internal/console"
	"github.com/peaberberian/paul-envs/internal/engine"
	"github.com/peaberberian/paul-envs/internal/files"
	"github.com/peaberberian/paul-envs/internal/utils"
)

func Stop(ctx context.Context, args []string, filestore *files.FileStore, console *console.Console) error {
	return terminateProject(ctx, args, filestore, console, "stop")
}

func Kill(ctx context.Context, args []string, filestore *files.FileStore, console *console.Console) error {
	return terminateProject(ctx, args, filestore, console, "kill")
}

func terminateProject(ctx context.Context, args []string, filestore *files.FileStore, console *console.Console, action string) error {
	flagset := newCommandFlagSet(action, console)
	var engineSelection string
	var timeout int
	flagset.StringVar(&engineSelection, "engine", "", "Container engine to use: docker or podman. Default: the last build engine for the project, otherwise auto-select.")
	description := "Immediately kill a project's active containers with SIGKILL."
	if action == "stop" {
		description = "Gracefully stop a project's active containers, then force termination if the timeout expires."
		flagset.IntVar(&timeout, "timeout", 0, "Seconds to wait before forcefully stopping; -1 waits indefinitely. Defaults to the container engine's configured timeout.")
	}
	flagset.Usage = func() {
		writeCommandUsage(console, flagset, "paul-envs "+action+" [flags] <project-name>",
			description+" All joined sessions exit. Project configuration, images, and persistent volumes are preserved.")
	}
	if err := parseCommandFlags(flagset, args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if flagset.NArg() != 1 {
		return fmt.Errorf("%s requires exactly one project name", action)
	}
	if timeout < -1 {
		return fmt.Errorf("--timeout must be -1 or greater")
	}
	name := flagset.Arg(0)
	if err := utils.ValidateProjectName(name); err != nil {
		return err
	}
	requestedEngine, err := parseCommandEngineSelection(engineSelection)
	if err != nil {
		return err
	}
	if !filestore.DoesProjectExist(name) {
		return fmt.Errorf("project '%s' not found\nHint: Use 'paul-envs list' to see available projects", name)
	}
	containerEngine, _, err := newProjectEngine(ctx, name, requestedEngine, filestore, console)
	if err != nil {
		return err
	}
	options := engine.StopOptions{}
	flagset.Visit(func(f *flag.Flag) {
		if f.Name == "timeout" {
			options.Timeout = &timeout
		}
	})
	return terminateProjectContainers(ctx, name, containerEngine, console, action, options)
}

func terminateProjectContainers(ctx context.Context, name string, containerEngine engine.ContainerEngine, console *console.Console, action string, options engine.StopOptions) error {
	containers, err := containerEngine.ListContainers(ctx)
	if err != nil {
		return fmt.Errorf("cannot list containers for project '%s': %w", name, err)
	}
	count := 0
	for _, container := range containers {
		if container.ProjectName == nil || *container.ProjectName != name {
			continue
		}
		switch container.State {
		case "running", "paused", "restarting", "stopping":
		case "created", "configured", "initialized", "exited", "stopped", "dead", "removing":
			continue
		default:
			return fmt.Errorf("cannot %s container '%s': unknown state %q", action, container.ContainerId, container.State)
		}
		if action == "kill" {
			err = containerEngine.KillContainer(ctx, container)
		} else {
			err = containerEngine.StopContainer(ctx, container, options)
		}
		if err != nil {
			return fmt.Errorf("failed to %s container '%s' for project '%s': %w", action, container.ContainerId, name, err)
		}
		count++
	}
	if count == 0 {
		console.Info("No active containers for project '%s'.", name)
	} else if action == "kill" {
		console.Success("Killed %d container(s) for project '%s'.", count, name)
	} else {
		console.Success("Stopped %d container(s) for project '%s'.", count, name)
	}
	return nil
}
