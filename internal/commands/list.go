package commands

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/peaberberian/paul-envs/internal/console"
	"github.com/peaberberian/paul-envs/internal/engine"
	"github.com/peaberberian/paul-envs/internal/files"
)

func List(ctx context.Context, args []string, filestore *files.FileStore, console *console.Console) error {
	nameOnly := false
	flagset := newCommandFlagSet("list", console)
	flagset.BoolVar(&nameOnly, "names", false, "Only display names")
	flagset.Usage = func() {
		writeCommandUsage(
			console,
			flagset,
			"paul-envs list [flags]",
			"List all projects and their current status.",
		)
	}
	if err := parseCommandFlags(flagset, args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	entries, err := filestore.GetAllProjects()
	if err != nil {
		return fmt.Errorf("could not list all projects: %w", err)
	}

	if len(entries) == 0 {
		console.WriteLn("  (no project found)")
		console.WriteLn("Hint: Create one with 'paul-envs create <path>'")
	} else if nameOnly {
		for _, entry := range entries {
			console.WriteLn(entry.ProjectName)
		}
	} else {
		engineCache := map[engine.Selection]engine.ContainerEngine{}
		var allEngines []engine.ContainerEngine

		for _, entry := range entries {
			imageInfo, warnErr := listProjectImageInfo(ctx, entry.ProjectName, filestore, console, engineCache, &allEngines)
			if warnErr != nil {
				console.Warn("Could not obtain image info for project '%s': %s", entry.ProjectName, warnErr)
			}
			printProjectInfo(entry, imageInfo, console)
		}
		if len(entries) <= 1 {
			console.WriteLn("Total: %d project", len(entries))
		} else {
			console.WriteLn("Total: %d projects", len(entries))
		}
	}
	return nil
}

func listProjectImageInfo(
	ctx context.Context,
	projectName string,
	filestore *files.FileStore,
	console *console.Console,
	engineCache map[engine.Selection]engine.ContainerEngine,
	allEngines *[]engine.ContainerEngine,
) (*engine.ImageInfo, error) {
	selectedEngine, err := listProjectEngineSelection(projectName, filestore)
	if err != nil {
		return nil, err
	}

	if selectedEngine != engine.SelectionAuto {
		containerEngine, ok := engineCache[selectedEngine]
		if !ok {
			containerEngine, err = engine.NewSelected(ctx, console, selectedEngine)
			if err != nil {
				return nil, err
			}
			engineCache[selectedEngine] = containerEngine
		}
		return containerEngine.GetImageInfo(ctx, projectName)
	}

	if *allEngines == nil {
		engines, err := engine.NewSet(ctx, console, engine.SelectionAll)
		if err != nil {
			return nil, err
		}
		*allEngines = engines
	}

	var lastErr error
	for _, containerEngine := range *allEngines {
		imageInfo, err := containerEngine.GetImageInfo(ctx, projectName)
		if err != nil {
			lastErr = err
			continue
		}
		if imageInfo != nil {
			return imageInfo, lastErr
		}
	}
	return nil, lastErr
}

func listProjectEngineSelection(projectName string, filestore *files.FileStore) (engine.Selection, error) {
	lastBuildEngine, err := filestore.GetBuildEngineSelection(projectName)
	if err != nil {
		if os.IsNotExist(err) || strings.Contains(err.Error(), "could not open 'project.buildinfo'") {
			return engine.SelectionAuto, nil
		}
		return engine.SelectionAuto, fmt.Errorf("could not determine the last build engine: %w", err)
	}

	selected, err := parseCommandEngineSelection(lastBuildEngine)
	if err != nil {
		return engine.SelectionAuto, fmt.Errorf("ignoring invalid build engine recorded in project metadata: %w", err)
	}
	return selected, nil
}

func printProjectInfo(projectEntry files.ProjectEntry, imageInfo *engine.ImageInfo, console *console.Console) bool {
	console.Info("%s", projectEntry.ProjectName)
	console.WriteLn("  Mounted project   : %s", projectEntry.ProjectPath)
	console.WriteLn("  build.conf file   : %s", projectEntry.BuildConfigPath)
	console.WriteLn("  run.conf file     : %s", projectEntry.RuntimeConfigPath)
	if imageInfo != nil {
		console.WriteLn("  Container image   : %s", imageInfo.ImageName)
		if imageInfo.BuiltAt == nil {
			console.WriteLn("  Last built at     : Never")
		} else {
			console.WriteLn("  Last built at     : %s", imageInfo.BuiltAt)
		}
	}
	console.WriteLn("")
	return true
}
