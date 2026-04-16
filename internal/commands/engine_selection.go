package commands

import (
	"context"
	"fmt"
	"os"

	"github.com/peaberberian/paul-envs/internal/console"
	"github.com/peaberberian/paul-envs/internal/engine"
	"github.com/peaberberian/paul-envs/internal/files"
)

func parseCommandEngineSelection(value string) (engine.Selection, error) {
	switch value {
	case "":
		return engine.SelectionAuto, nil
	case string(engine.SelectionDocker):
		return engine.SelectionDocker, nil
	case string(engine.SelectionPodman):
		return engine.SelectionPodman, nil
	default:
		return "", fmt.Errorf("invalid --engine value %q. Must be one of: docker, podman", value)
	}
}

func resolveProjectEngineSelection(
	projectName string,
	requested engine.Selection,
	filestore *files.FileStore,
	console *console.Console,
) engine.Selection {
	if requested != engine.SelectionAuto {
		return requested
	}

	lastBuildEngine, err := filestore.GetBuildEngineSelection(projectName)
	if err != nil {
		if !os.IsNotExist(err) {
			console.Warn("Could not determine the last build engine for '%s': %s", projectName, err)
		}
		return engine.SelectionAuto
	}
	selected, err := parseCommandEngineSelection(lastBuildEngine)
	if err != nil {
		console.Warn("Ignoring invalid engine recorded in build metadata for '%s': %s", projectName, err)
		return engine.SelectionAuto
	}
	if selected == engine.SelectionAuto {
		return engine.SelectionAuto
	}

	console.Info("Using the last build engine for '%s': %s.", projectName, selected)
	return selected
}

func buildArgsForEngine(projectName string, selection engine.Selection) []string {
	args := []string{}
	if selection != engine.SelectionAuto {
		args = append(args, "--engine", string(selection))
	}
	args = append(args, projectName)
	return args
}

func newProjectEngine(
	ctx context.Context,
	projectName string,
	requested engine.Selection,
	filestore *files.FileStore,
	console *console.Console,
) (engine.ContainerEngine, engine.Selection, error) {
	selected := resolveProjectEngineSelection(projectName, requested, filestore, console)
	containerEngine, err := engine.NewSelected(ctx, console, selected)
	if err != nil {
		return nil, engine.SelectionAuto, err
	}
	return containerEngine, selected, nil
}
