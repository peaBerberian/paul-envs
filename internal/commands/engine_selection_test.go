package commands

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	versions "github.com/peaberberian/paul-envs/internal"
	"github.com/peaberberian/paul-envs/internal/console"
	"github.com/peaberberian/paul-envs/internal/engine"
	"github.com/peaberberian/paul-envs/internal/files"
)

func TestParseCommandEngineSelection(t *testing.T) {
	tests := []struct {
		input string
		want  engine.Selection
		ok    bool
	}{
		{input: "", want: engine.SelectionAuto, ok: true},
		{input: "docker", want: engine.SelectionDocker, ok: true},
		{input: "podman", want: engine.SelectionPodman, ok: true},
		{input: "all", ok: false},
	}

	for _, tt := range tests {
		got, err := parseCommandEngineSelection(tt.input)
		if tt.ok && err != nil {
			t.Fatalf("parseCommandEngineSelection(%q) unexpected error: %v", tt.input, err)
		}
		if !tt.ok && err == nil {
			t.Fatalf("parseCommandEngineSelection(%q) expected error, got none", tt.input)
		}
		if tt.ok && got != tt.want {
			t.Fatalf("parseCommandEngineSelection(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestResolveProjectEngineSelectionUsesLastBuildEngine(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	store, err := files.NewFileStore()
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}

	projectName := "alpha"
	projectDir := filepath.Dir(store.GetProjectBuildInfoPath(projectName))
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("mkdir project dir: %v", err)
	}
	buildInfoPath := store.GetProjectBuildInfoPath(projectName)
	buildInfo := "" +
		"VERSION=" + versions.BuildInfoVersion.ToString() + "\n" +
		"BUILT_BY=test-machine\n" +
		"BUILD_CONFIG=test-hash\n" +
		"BUILD_CONFIG_VERSION=1.0.0\n" +
		"RUNTIME_CONFIG_VERSION=1.0.0\n" +
		"LAST_BUILT_AT=2026-04-16T12:00:00Z\n" +
		"CONTAINER_ENGINE=docker\n" +
		"CONTAINER_ENGINE_VERSION=27.0.0\n"
	if err := os.WriteFile(buildInfoPath, []byte(buildInfo), 0644); err != nil {
		t.Fatalf("write project.buildinfo: %v", err)
	}

	var out bytes.Buffer
	cons := console.New(context.Background(), strings.NewReader(""), &out, &out)

	got := resolveProjectEngineSelection(projectName, engine.SelectionAuto, store, cons)
	if got != engine.SelectionDocker {
		t.Fatalf("resolveProjectEngineSelection() = %q, want %q", got, engine.SelectionDocker)
	}
	if !strings.Contains(out.String(), "Using the last build engine") {
		t.Fatalf("expected informational output, got %q", out.String())
	}
}

func TestBuildArgsForEngine(t *testing.T) {
	got := buildArgsForEngine("alpha", engine.SelectionDocker)
	want := []string{"--engine", "docker", "alpha"}
	if strings.Join(got, "\x00") != strings.Join(want, "\x00") {
		t.Fatalf("buildArgsForEngine() = %v, want %v", got, want)
	}

	got = buildArgsForEngine("alpha", engine.SelectionAuto)
	want = []string{"alpha"}
	if strings.Join(got, "\x00") != strings.Join(want, "\x00") {
		t.Fatalf("buildArgsForEngine() = %v, want %v", got, want)
	}
}

func TestBuildHelpIncludesEngineFlag(t *testing.T) {
	var out bytes.Buffer
	cons := console.New(context.Background(), strings.NewReader(""), &out, &out)

	if err := Build(context.Background(), []string{"--help"}, nil, cons); err != nil {
		t.Fatalf("Build(--help) error = %v", err)
	}

	got := out.String()
	for _, fragment := range []string{
		"Usage: paul-envs build [project-name] [flags]",
		"--engine string",
		"Container engine to use for this build: docker or podman.",
	} {
		if !strings.Contains(got, fragment) {
			t.Fatalf("expected help output to contain %q, got:\n%s", fragment, got)
		}
	}
}

func TestRunHelpIncludesEngineFlag(t *testing.T) {
	var out bytes.Buffer
	cons := console.New(context.Background(), strings.NewReader(""), &out, &out)

	if err := Run(context.Background(), []string{"--help"}, nil, cons); err != nil {
		t.Fatalf("Run(--help) error = %v", err)
	}

	got := out.String()
	for _, fragment := range []string{
		"Usage: paul-envs run [project-name] [command...] [flags]",
		"--engine string",
		"Container engine to use: docker or podman.",
	} {
		if !strings.Contains(got, fragment) {
			t.Fatalf("expected help output to contain %q, got:\n%s", fragment, got)
		}
	}
}

func TestRemoveHelpIncludesEngineFlag(t *testing.T) {
	var out bytes.Buffer
	cons := console.New(context.Background(), strings.NewReader(""), &out, &out)

	if err := Remove(context.Background(), []string{"--help"}, nil, cons); err != nil {
		t.Fatalf("Remove(--help) error = %v", err)
	}

	got := out.String()
	for _, fragment := range []string{
		"Usage: paul-envs remove [flags] [project-name]",
		"--engine string",
		"Container engine to use for asset removal: docker or podman.",
	} {
		if !strings.Contains(got, fragment) {
			t.Fatalf("expected help output to contain %q, got:\n%s", fragment, got)
		}
	}
}
