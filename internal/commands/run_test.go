package commands

import (
	"context"
	"testing"

	"github.com/peaberberian/paul-envs/internal/engine"
	"github.com/peaberberian/paul-envs/internal/files"
)

type stubEngine struct {
	name    string
	version string
}

func (s *stubEngine) Info(context.Context) (engine.EngineInfo, error) {
	return engine.EngineInfo{Name: s.name, Version: s.version}, nil
}

func (s *stubEngine) BuildImage(context.Context, files.ProjectEntry, engine.BuildOptions) error {
	return nil
}

func (s *stubEngine) RunContainer(context.Context, files.ProjectEntry, []string) error {
	return nil
}

func (s *stubEngine) JoinContainer(context.Context, engine.ContainerInfo, []string) error {
	return nil
}

func (s *stubEngine) CreateVolume(context.Context, string) error {
	return nil
}

func (s *stubEngine) HasBeenBuilt(context.Context, string) (bool, error) {
	return false, nil
}

func (s *stubEngine) GetImageInfo(context.Context, string) (*engine.ImageInfo, error) {
	return nil, nil
}

func (s *stubEngine) ListContainers(context.Context) ([]engine.ContainerInfo, error) {
	return []engine.ContainerInfo{}, nil
}

func (s *stubEngine) RemoveContainer(context.Context, engine.ContainerInfo) error {
	return nil
}

func (s *stubEngine) ListImages(context.Context) ([]engine.ImageInfo, error) {
	return []engine.ImageInfo{}, nil
}

func (s *stubEngine) RemoveImage(context.Context, engine.ImageInfo) error {
	return nil
}

func (s *stubEngine) ListVolumes(context.Context) ([]engine.VolumeInfo, error) {
	return []engine.VolumeInfo{}, nil
}

func (s *stubEngine) RemoveVolume(context.Context, engine.VolumeInfo) error {
	return nil
}

func (s *stubEngine) ListNetworks(context.Context) ([]engine.NetworkInfo, error) {
	return []engine.NetworkInfo{}, nil
}

func (s *stubEngine) RemoveNetwork(context.Context, engine.NetworkInfo) error {
	return nil
}

func (s *stubEngine) PruneBuildCache(context.Context) error {
	return nil
}

func (s *stubEngine) SupportsBuildCachePrune() bool {
	return false
}

func TestRunRebuildDecisionDetectsEngineSwitch(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	store, err := files.NewFileStore()
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}

	projectPath := t.TempDir()
	if err := store.CreateProjectFiles(
		"switch-engine",
		files.BuildTemplateData{
			Version:           "1.0.0",
			HostUID:           "1000",
			HostGID:           "1000",
			Username:          "dev",
			Shell:             "bash",
			InstallNode:       "none",
			InstallRust:       "none",
			InstallPython:     "none",
			InstallGo:         "none",
			EnableWasm:        "false",
			EnableSSH:         "false",
			EnableSudo:        "false",
			Packages:          "",
			InstallNeovim:     "false",
			InstallStarship:   "false",
			InstallOhMyPosh:   "false",
			InstallAtuin:      "false",
			InstallMise:       "false",
			InstallZellij:     "false",
			InstallJujutsu:    "false",
			InstallDelta:      "false",
			InstallOpenCode:   "false",
			InstallClaudeCode: "false",
			InstallCodex:      "false",
			InstallFirefox:    "false",
		},
		files.RuntimeTemplateData{
			Version:         "1.0.0",
			ProjectHostPath: projectPath,
		},
	); err != nil {
		t.Fatalf("CreateProjectFiles() error = %v", err)
	}

	if err := store.RefreshBuildInfoFile("switch-engine", "docker", "27.0.0"); err != nil {
		t.Fatalf("RefreshBuildInfoFile(docker) error = %v", err)
	}

	needsRebuild, reason, err := runRebuildDecision(
		context.Background(),
		"switch-engine",
		store,
		&stubEngine{name: "podman", version: "5.0.0"},
	)
	if err != nil {
		t.Fatalf("runRebuildDecision() error = %v", err)
	}
	if !needsRebuild {
		t.Fatalf("runRebuildDecision() needsRebuild = false, want true")
	}
	if reason != files.RebuildDifferentEngine {
		t.Fatalf("runRebuildDecision() reason = %v, want %v", reason, files.RebuildDifferentEngine)
	}
}
