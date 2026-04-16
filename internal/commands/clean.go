package commands

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"strings"

	"github.com/peaberberian/paul-envs/internal/console"
	"github.com/peaberberian/paul-envs/internal/engine"
	"github.com/peaberberian/paul-envs/internal/files"
)

func Clean(ctx context.Context, args []string, filestore *files.FileStore, console *console.Console) error {
	var noPrompt bool
	var engineSelection string
	var projectsOnly bool
	var configOnly bool
	var resourcesOnly bool
	var buildCacheOnly bool
	flagset := newCommandFlagSet("clean", console)
	flagset.BoolVar(&noPrompt, "no-prompt", false, "Non-interactive mode: apply the default answers to each cleanup step")
	flagset.StringVar(&engineSelection, "engine", "", "Container engine to clean: docker, podman, or all. Default: the selected engine for this run.")
	flagset.BoolVar(&projectsOnly, "projects", false, "Only remove stored project configuration files")
	flagset.BoolVar(&configOnly, "config", false, "Only remove the global paul-envs configuration")
	flagset.BoolVar(&resourcesOnly, "managed-resources", false, "Only remove managed containers, images, volumes, and networks")
	flagset.BoolVar(&buildCacheOnly, "build-cache", false, "Only prune cached build data associated with paul-envs images")
	flagset.Usage = func() {
		writeCommandUsage(
			console,
			flagset,
			"paul-envs clean [flags]",
			"Remove global paul-envs data, project configurations, and managed container assets across projects.",
		)
	}
	if err := parseCommandFlags(flagset, args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	selectedEngine, err := parseCleanEngineSelection(engineSelection)
	if err != nil {
		return err
	}

	cleanOpts := newCleanOptions(projectsOnly, configOnly, resourcesOnly, buildCacheOnly)

	if cleanOpts.projects {
		console.Info("\n1. Projects' configuration")
		console.WriteLn("This will clean-up the container configurations you created with the 'create' command.")
		choice, err := yesNoWithOptionalPrompt(console, noPrompt, "Remove projects configuration files?", true)
		if err != nil {
			return err
		} else if !choice {
			console.WriteLn("\nSkipping container configurations")
		} else {
			console.WriteLn("\nRemoving projects configuration files...")
			if err := filestore.DeleteDataDirectory(); err != nil {
				return err
			}
		}
	}

	if cleanOpts.config {
		console.Info("\n2. paul-envs' configuration")
		console.WriteLn("This will reset the global 'paul-envs' configuration.")
		choice, err := yesNoWithOptionalPrompt(console, noPrompt, "Remove paul-envs configuration?", true)
		if err != nil {
			return err
		} else if !choice {
			console.WriteLn("\nSkipping paul-envs configuration")
		} else {
			console.WriteLn("\nRemoving paul-envs configuration...")
			if err := filestore.DeleteConfigDirectory(); err != nil {
				return err
			}
		}
	}

	var containerEngines []engine.ContainerEngine
	if cleanOpts.managedResources || cleanOpts.buildCache {
		containerEngines, err = engine.NewSet(ctx, console, selectedEngine)
		if err != nil {
			return err
		}
		describeSelectedEngines(console, containerEngines, selectedEngine)
	}

	if cleanOpts.managedResources {
		console.Info("\n3. Managed paul-envs resources")
		console.WriteLn("This will remove containers, images, volumes, and networks created by paul-envs.")
		choice, err := yesNoWithOptionalPrompt(console, noPrompt, "Remove managed resources?", true)
		if err != nil {
			return err
		} else if !choice {
			console.WriteLn("\nSkipping managed resource removal")
		} else {
			if err := removeManagedResources(ctx, containerEngines, console); err != nil {
				return err
			}
		}
	}

	if cleanOpts.buildCache {
		buildCacheEngines := enginesSupportingBuildCachePrune(containerEngines)
		unsupportedBuildCacheEngines := enginesWithoutBuildCachePrune(containerEngines)
		if len(buildCacheEngines) == 0 {
			console.Info("\n4. Builder cache")
			console.WriteLn("Scoped builder-cache pruning is not supported for: %s.", strings.Join(engineNames(unsupportedBuildCacheEngines), ", "))
		} else {
			console.Info("\n4. Builder cache")
			console.WriteLn("This will remove cached build data associated with paul-envs images.")
			console.WriteLn("Future builds may be slower.")
			if len(unsupportedBuildCacheEngines) > 0 {
				console.WriteLn("Scoped builder-cache pruning will be skipped for: %s.", strings.Join(engineNames(unsupportedBuildCacheEngines), ", "))
			}
			choice, err := yesNoWithOptionalPrompt(console, noPrompt, "Prune builder cache?", false)
			if err != nil {
				return err
			} else if !choice {
				console.WriteLn("\nSkipping builder cache pruning")
			} else if err := pruneBuildCache(ctx, buildCacheEngines, console); err != nil {
				return err
			}
		}
	}

	console.Success("\nCleanup complete!")
	return nil
}

type cleanOptions struct {
	projects         bool
	config           bool
	managedResources bool
	buildCache       bool
}

func newCleanOptions(projects, config, managedResources, buildCache bool) cleanOptions {
	if !projects && !config && !managedResources && !buildCache {
		return cleanOptions{
			projects:         true,
			config:           true,
			managedResources: true,
			buildCache:       true,
		}
	}
	return cleanOptions{
		projects:         projects,
		config:           config,
		managedResources: managedResources,
		buildCache:       buildCache,
	}
}

func parseCleanEngineSelection(value string) (engine.Selection, error) {
	switch value {
	case "":
		return engine.SelectionAuto, nil
	case string(engine.SelectionDocker):
		return engine.SelectionDocker, nil
	case string(engine.SelectionPodman):
		return engine.SelectionPodman, nil
	case string(engine.SelectionAll):
		return engine.SelectionAll, nil
	default:
		return "", fmt.Errorf("invalid --engine value %q. Must be one of: docker, podman, all", value)
	}
}

func yesNoWithOptionalPrompt(console *console.Console, noPrompt bool, prompt string, defaultVal bool) (bool, error) {
	if !noPrompt {
		return console.AskYesNo(prompt, defaultVal)
	}
	console.Info("Using default answer for '%s': %t", prompt, defaultVal)
	return defaultVal, nil
}

func describeSelectedEngines(console *console.Console, containerEngines []engine.ContainerEngine, selection engine.Selection) {
	engineNames := engineNames(containerEngines)

	switch selection {
	case engine.SelectionAll:
		console.Info("Cleaning all available engines: %s.", strings.Join(engineNames, ", "))
	case engine.SelectionDocker, engine.SelectionPodman:
		console.Info("Cleaning container engine: %s.", strings.Join(engineNames, ", "))
	default:
		if len(engineNames) > 1 {
			console.Info("Cleaning selected container engines: %s.", strings.Join(engineNames, ", "))
		} else if len(engineNames) == 1 {
			console.Info("Cleaning selected container engine: %s.", engineNames[0])
		}
	}
}

func removeManagedResources(ctx context.Context, containerEngines []engine.ContainerEngine, console *console.Console) error {
	for _, containerEngine := range containerEngines {
		writeEngineSection(console, containerEngine)
		if err := removeContainers(ctx, containerEngine, console); err != nil {
			return err
		}
		if err := removeImages(ctx, containerEngine, console); err != nil {
			return err
		}
		if err := removeVolumes(ctx, containerEngine, console); err != nil {
			return err
		}
		if err := removeNetworks(ctx, containerEngine, console); err != nil {
			return err
		}
	}
	return nil
}

func removeContainers(ctx context.Context, containerEngine engine.ContainerEngine, console *console.Console) error {
	console.WriteLn("\nStopping and removing containers...")

	containers, err := containerEngine.ListContainers(ctx)
	if err != nil {
		return fmt.Errorf("cannot list current containers: %w", err)
	}
	for _, container := range containers {
		if container.ContainerName == nil {
			console.WriteLn("  • Removing unknown container")
		} else {
			console.WriteLn("  • Removing container: %s", *container.ContainerName)
		}
		if err := containerEngine.RemoveContainer(ctx, container); err != nil {
			console.Warn("    WARNING: failed to remove container: %v", err)
		}
	}
	return nil
}

func removeImages(ctx context.Context, containerEngine engine.ContainerEngine, console *console.Console) error {
	console.WriteLn("\nRemoving images...")

	images, err := containerEngine.ListImages(ctx)
	if err != nil {
		return fmt.Errorf("cannot list current images: %w", err)
	}
	for _, image := range images {
		console.WriteLn("  • Removing image: %s", image.ImageName)
		if err := containerEngine.RemoveImage(ctx, image); err != nil {
			console.Warn("    WARNING: failed to remove image: %v", err)
		}
	}
	return nil
}

func removeVolumes(ctx context.Context, containerEngine engine.ContainerEngine, console *console.Console) error {
	console.WriteLn("\nRemoving volumes...")
	volumes, err := containerEngine.ListVolumes(ctx)
	if err != nil {
		return fmt.Errorf("cannot list current volumes: %w", err)
	}
	for _, volume := range volumes {
		console.WriteLn("  • Removing volume: %s", volume.VolumeName)
		if err := containerEngine.RemoveVolume(ctx, volume); err != nil {
			console.Warn("    WARNING: failed to remove volume: %v", err)
		}
	}
	return nil
}

func removeNetworks(ctx context.Context, containerEngine engine.ContainerEngine, console *console.Console) error {
	console.WriteLn("\nRemoving networks...")
	networks, err := containerEngine.ListNetworks(ctx)
	if err != nil {
		return fmt.Errorf("cannot list current networks: %w", err)
	}
	for _, network := range networks {
		console.WriteLn("  • Removing network: %s", network.NetworkName)
		if err := containerEngine.RemoveNetwork(ctx, network); err != nil {
			console.Warn("    WARNING: failed to remove network: %v", err)
		}
	}
	return nil
}

func enginesSupportingBuildCachePrune(containerEngines []engine.ContainerEngine) []engine.ContainerEngine {
	result := make([]engine.ContainerEngine, 0, len(containerEngines))
	for _, containerEngine := range containerEngines {
		if containerEngine.SupportsBuildCachePrune() {
			result = append(result, containerEngine)
		}
	}
	return result
}

func enginesWithoutBuildCachePrune(containerEngines []engine.ContainerEngine) []engine.ContainerEngine {
	result := make([]engine.ContainerEngine, 0, len(containerEngines))
	for _, containerEngine := range containerEngines {
		if !containerEngine.SupportsBuildCachePrune() {
			result = append(result, containerEngine)
		}
	}
	return result
}

func pruneBuildCache(ctx context.Context, containerEngines []engine.ContainerEngine, console *console.Console) error {
	for _, containerEngine := range containerEngines {
		writeEngineSection(console, containerEngine)
		console.WriteLn("\nPruning build cache...")
		if err := containerEngine.PruneBuildCache(ctx); err != nil {
			return err
		}
	}
	return nil
}

func writeEngineSection(console *console.Console, containerEngine engine.ContainerEngine) {
	console.Info("\nEngine: %s", cleanEngineName(containerEngine))
}

func cleanEngineName(containerEngine engine.ContainerEngine) string {
	switch containerEngine.(type) {
	case *engine.DockerEngine:
		return "docker"
	case *engine.PodmanEngine:
		return "podman"
	default:
		return "unknown"
	}
}

func engineNames(containerEngines []engine.ContainerEngine) []string {
	names := make([]string, 0, len(containerEngines))
	for _, containerEngine := range containerEngines {
		names = append(names, cleanEngineName(containerEngine))
	}
	return names
}
