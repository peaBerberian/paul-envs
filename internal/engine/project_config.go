package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/peaberberian/paul-envs/internal/config"
	"github.com/peaberberian/paul-envs/internal/files"
)

func loadBuildConfig(project files.ProjectEntry) (config.BuildConfig, error) {
	cfg, err := config.LoadBuildConfig(project.BuildConfigPath)
	if err != nil {
		return config.BuildConfig{}, fmt.Errorf("failed to load build config for project %q: %w", project.ProjectName, err)
	}
	return cfg, nil
}

func loadRuntimeConfig(project files.ProjectEntry) (config.RuntimeConfig, error) {
	cfg, err := config.LoadRuntimeConfig(project.RuntimeConfigPath)
	if err != nil {
		return config.RuntimeConfig{}, fmt.Errorf("failed to load runtime config for project %q: %w", project.ProjectName, err)
	}
	return cfg, nil
}

func projectBaseDataDir(project files.ProjectEntry) string {
	return filepath.Clean(filepath.Join(filepath.Dir(project.BuildConfigPath), "..", ".."))
}

func projectDockerfilePath(project files.ProjectEntry) string {
	return filepath.Join(projectBaseDataDir(project), "Dockerfile")
}

func projectImageName(projectName string) string {
	return fmt.Sprintf("paulenv:%s", projectName)
}

func projectContainerName(projectName string) string {
	return fmt.Sprintf("paulenv-%s", projectName)
}

func projectLocalVolumeName(projectName string) string {
	return fmt.Sprintf("paulenv-%s-local", projectName)
}

func projectMountTarget(username, projectName string) string {
	return fmt.Sprintf("/home/%s/projects/%s", username, projectName)
}

func resolveRuntimePath(configPath, configuredPath string) (string, error) {
	if configuredPath == "" {
		return "", nil
	}
	if filepath.IsAbs(configuredPath) {
		return filepath.Clean(configuredPath), nil
	}
	if configuredPath == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		return home, nil
	}
	if strings.HasPrefix(configuredPath, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		return filepath.Join(home, configuredPath[2:]), nil
	}
	return filepath.Clean(filepath.Join(filepath.Dir(configPath), configuredPath)), nil
}
