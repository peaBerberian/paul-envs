package commands

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"testing/iotest"

	versions "github.com/peaberberian/paul-envs/internal"
	"github.com/peaberberian/paul-envs/internal/console"
	"github.com/peaberberian/paul-envs/internal/engine"
	"github.com/peaberberian/paul-envs/internal/files"
)

func TestRunRebuildConfirmationReadError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake engines require /bin/sh")
	}
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	store, err := files.NewFileStore()
	if err != nil {
		t.Fatal(err)
	}
	lockPath := filepath.Join(filepath.Dir(store.GetProjectBuildInfoPath("alpha")), "project.lock")
	for path, content := range map[string]string{
		store.GetProjectBuildConfigPath("alpha"):   "VERSION 1.0.0\nHOST_UID 1000\nHOST_GID 1000\nUSERNAME dev\nUSER_SHELL bash\n",
		store.GetProjectRuntimeConfigPath("alpha"): "VERSION 1.0.0\nPATH /tmp/alpha\n",
		lockPath: "VERSION=" + versions.ProjectLockVersion.ToString() + "\nDOCKERFILE_VERSION=" + versions.DockerfileVersion.ToString() + "\n",
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	dir := t.TempDir()
	logPath := filepath.Join(dir, "engine-calls")
	script := `#!/bin/sh
printf '%s\n' "$*" >> "$PAUL_ENVS_TEST_ENGINE_CALLS"
case "$1" in
    --version)
        case "${0##*/}" in
            docker) printf 'Docker version 27.0.0\n' ;;
            podman) printf 'podman version 5.0.0\n' ;;
        esac
        ;;
    *) exit 1 ;;
esac
`
	for _, binary := range []string{"docker", "podman"} {
		if err := os.WriteFile(filepath.Join(dir, binary), []byte(script), 0700); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", dir)
	t.Setenv("PAUL_ENVS_TEST_ENGINE_CALLS", logPath)
	for _, binary := range []string{"docker", "podman"} {
		for _, readErr := range []error{io.EOF, errors.New("input unavailable")} {
			t.Run(binary+"/"+readErr.Error(), func(t *testing.T) {
				previousEngine := "docker"
				if binary == "docker" {
					previousEngine = "podman"
				}
				if err := store.RefreshBuildInfoFile("alpha", previousEngine, "1.0.0"); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(logPath, nil, 0600); err != nil {
					t.Fatal(err)
				}
				var out bytes.Buffer
				ctx := context.Background()
				cons := console.New(ctx, iotest.ErrReader(readErr), &out, &out)
				err := Run(ctx, []string{"--engine", binary, "alpha"}, store, cons)
				if !errors.Is(err, readErr) {
					t.Errorf("Run() error = %v, want %v", err, readErr)
				}
				if !strings.Contains(out.String(), "Do you want to build it first?") {
					t.Errorf("rebuild confirmation not reached: %s", &out)
				}
				calls, err := os.ReadFile(logPath)
				if err != nil {
					t.Fatal(err)
				}
				if string(calls) != "--version\n" {
					t.Errorf("unexpected engine calls after confirmation error: %s", calls)
				}
			})
		}
	}
}

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

func (s *stubEngine) StopContainer(context.Context, engine.ContainerInfo, engine.StopOptions) error {
	return nil
}

func (s *stubEngine) KillContainer(context.Context, engine.ContainerInfo) error {
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

func (s *stubEngine) GetStorageUsage(context.Context, []string) (engine.StorageUsage, error) {
	return engine.StorageUsage{}, nil
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
