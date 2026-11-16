package commands

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"slices"
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
		engineCache := map[engine.Selection]*listEngineStatus{}
		projectNames := make([]string, len(entries))
		for i, entry := range entries {
			projectNames[i] = entry.ProjectName
		}
		storageCache := map[engine.Selection]engine.StorageUsage{}

		for _, entry := range entries {
			info, warnErr := listProjectInfo(ctx, entry.ProjectName, filestore, console, engineCache)
			if warnErr != nil {
				console.Warn("Could not obtain status for project '%s': %s", entry.ProjectName, warnErr)
			}
			selection := engine.Selection(info.engineName)
			if status := engineCache[selection]; status != nil && status.containerEngine != nil {
				storage, queried := storageCache[selection]
				if !queried {
					if status.err == nil {
						var err error
						storage, err = status.containerEngine.GetStorageUsage(ctx, projectNames)
						if err != nil {
							console.Warn("Could not obtain storage usage for %s: %s", selection, err)
						}
					}
					storageCache[selection] = storage
				}
				info.storage = storage.Projects[entry.ProjectName]
			}
			printProjectInfo(entry, info, console)
		}
		for _, selection := range []engine.Selection{engine.SelectionPodman, engine.SelectionDocker} {
			if storage, ok := storageCache[selection]; ok {
				value := formatStorageSize(storage.SharedCacheBytes)
				if len(storageCache) > 1 {
					value += " (" + string(selection) + ")"
				}
				console.WriteLn("%-21s : %s", "Shared cache size", value)
			}
		}
		if len(entries) <= 1 {
			console.WriteLn("Total: %d project", len(entries))
		} else {
			console.WriteLn("Total: %d projects", len(entries))
		}
	}
	return nil
}

type projectListInfo struct {
	image        *engine.ImageInfo
	engineName   string
	runtimeState string
	storage      engine.ProjectStorageUsage
}

type listEngineStatus struct {
	containerEngine engine.ContainerEngine
	containers      []engine.ContainerInfo
	err             error
}

func listProjectInfo(
	ctx context.Context,
	projectName string,
	filestore *files.FileStore,
	console *console.Console,
	engineCache map[engine.Selection]*listEngineStatus,
) (projectListInfo, error) {
	selectedEngine, err := listProjectEngineSelection(projectName, filestore)
	if err != nil {
		return projectListInfo{engineName: "unknown", runtimeState: "unknown"}, err
	}

	if selectedEngine != engine.SelectionAuto {
		status := getListEngineStatus(ctx, selectedEngine, console, engineCache)
		return inspectListProject(ctx, projectName, selectedEngine, status)
	}

	fallback := projectListInfo{engineName: "unknown", runtimeState: "unknown"}
	var lookupErr error
	available := false
	for _, selection := range []engine.Selection{engine.SelectionPodman, engine.SelectionDocker} {
		status := getListEngineStatus(ctx, selection, console, engineCache)
		if status.containerEngine == nil {
			continue
		}
		available = true
		info, err := inspectListProject(ctx, projectName, selection, status)
		if (info.image != nil && info.image.BuiltAt != nil) ||
			(info.runtimeState != "not running" && info.runtimeState != "unknown") {
			return info, err
		}
		lookupErr = errors.Join(lookupErr, err)
		if fallback.engineName == "unknown" && err == nil {
			fallback = info
		}
	}
	if !available {
		return fallback, errors.New("no supported container engine found, please install podman or docker first")
	}
	if lookupErr != nil {
		return projectListInfo{engineName: "unknown", runtimeState: "unknown"}, lookupErr
	}
	return fallback, nil
}

func getListEngineStatus(ctx context.Context, selection engine.Selection, console *console.Console, cache map[engine.Selection]*listEngineStatus) *listEngineStatus {
	if status, ok := cache[selection]; ok {
		return status
	}
	containerEngine, err := engine.NewSelected(ctx, console, selection)
	status := &listEngineStatus{containerEngine: containerEngine, err: err}
	if err == nil {
		status.containers, status.err = containerEngine.ListContainers(ctx)
	}
	cache[selection] = status
	return status
}

func inspectListProject(ctx context.Context, projectName string, selection engine.Selection, status *listEngineStatus) (projectListInfo, error) {
	info := projectListInfo{engineName: string(selection), runtimeState: "unknown"}
	if status.containerEngine == nil {
		return info, status.err
	}
	var imageErr error
	info.image, imageErr = status.containerEngine.GetImageInfo(ctx, projectName)
	if status.err == nil {
		info.runtimeState = projectRuntimeState(projectName, status.containers)
	}
	if imageErr != nil && status.err != nil && imageErr.Error() == status.err.Error() {
		return info, imageErr
	}
	return info, errors.Join(imageErr, status.err)
}

func projectRuntimeState(projectName string, containers []engine.ContainerInfo) string {
	var states []string
	for _, container := range containers {
		if container.ProjectName == nil || *container.ProjectName != projectName {
			continue
		}
		state := container.State
		switch state {
		case "exited", "stopped":
			state = "stopped"
		case "configured", "initialized":
			state = "created"
		case "":
			state = "unknown"
		}
		if !slices.Contains(states, state) {
			states = append(states, state)
		}
	}
	if len(states) == 0 {
		return "not running"
	}
	slices.Sort(states)
	return strings.Join(states, ", ")
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

func printProjectInfo(projectEntry files.ProjectEntry, info projectListInfo, console *console.Console) bool {
	field := func(label, value string) {
		console.WriteLn("  %-19s : %s", label, value)
	}
	console.Info("%s", projectEntry.ProjectName)
	field("Mounted project", projectEntry.ProjectPath)
	field("Status", info.runtimeState)
	field("build.conf file", projectEntry.BuildConfigPath)
	field("run.conf file", projectEntry.RuntimeConfigPath)
	var imageBytes *int64
	if imageInfo := info.image; imageInfo != nil {
		imageBytes = imageInfo.SizeBytes
		field("Container image", fmt.Sprintf("%s (%s)", imageInfo.ImageName, info.engineName))
		if imageInfo.BuiltAt == nil {
			field("Last built at", "Unknown")
		} else {
			field("Last built at", fmt.Sprint(imageInfo.BuiltAt))
		}
	}
	field("Image size", formatStorageSize(imageBytes))
	field("Persisted data size", formatStorageSize(info.storage.PersistedBytes))
	field("Temporary data size", formatStorageSize(info.storage.TemporaryBytes))
	console.WriteLn("")
	return true
}

func formatStorageSize(size *int64) string {
	if size == nil || *size < 0 {
		return "Unknown"
	}
	if *size < 1024 {
		return fmt.Sprintf("%d B", *size)
	}
	value := float64(*size)
	for _, unit := range []string{"KiB", "MiB", "GiB", "TiB", "PiB", "EiB"} {
		value /= 1024
		if value < 1024 || unit == "EiB" {
			return fmt.Sprintf("%.1f %s", value, unit)
		}
	}
	return "Unknown"
}
