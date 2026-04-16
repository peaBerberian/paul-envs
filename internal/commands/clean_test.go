package commands

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/peaberberian/paul-envs/internal/console"
	"github.com/peaberberian/paul-envs/internal/engine"
	"github.com/peaberberian/paul-envs/internal/files"
)

func TestYesNoWithOptionalPrompt_NoPromptUsesDefault(t *testing.T) {
	out := &bytes.Buffer{}
	cons := console.New(context.Background(), strings.NewReader("n\n"), out, &bytes.Buffer{})

	got, err := yesNoWithOptionalPrompt(cons, true, "Remove cache?", false)
	if err != nil {
		t.Fatalf("yesNoWithOptionalPrompt() error = %v", err)
	}
	if got {
		t.Fatalf("yesNoWithOptionalPrompt() = %v, want false", got)
	}
	if !strings.Contains(out.String(), "Using default answer") {
		t.Fatalf("expected informational output, got %q", out.String())
	}
}

func TestParseCleanEngineSelection(t *testing.T) {
	tests := []struct {
		input string
		want  engine.Selection
		ok    bool
	}{
		{input: "", want: engine.SelectionAuto, ok: true},
		{input: "docker", want: engine.SelectionDocker, ok: true},
		{input: "podman", want: engine.SelectionPodman, ok: true},
		{input: "all", want: engine.SelectionAll, ok: true},
		{input: "nope", ok: false},
	}

	for _, tt := range tests {
		got, err := parseCleanEngineSelection(tt.input)
		if tt.ok && err != nil {
			t.Fatalf("parseCleanEngineSelection(%q) unexpected error: %v", tt.input, err)
		}
		if !tt.ok && err == nil {
			t.Fatalf("parseCleanEngineSelection(%q) expected error, got none", tt.input)
		}
		if tt.ok && got != tt.want {
			t.Fatalf("parseCleanEngineSelection(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestEnginesSupportingBuildCachePrune(t *testing.T) {
	engines := []engine.ContainerEngine{
		&engine.PodmanEngine{},
		&engine.DockerEngine{},
	}

	got := enginesSupportingBuildCachePrune(engines)
	if len(got) != 1 {
		t.Fatalf("enginesSupportingBuildCachePrune() returned %d engines, want 1", len(got))
	}
	if _, ok := got[0].(*engine.DockerEngine); !ok {
		t.Fatalf("enginesSupportingBuildCachePrune() kept %T, want DockerEngine", got[0])
	}
}

func TestNewCleanOptions_DefaultsToAllSteps(t *testing.T) {
	got := newCleanOptions(false, false, false, false)
	if !got.projects || !got.config || !got.managedResources || !got.buildCache {
		t.Fatalf("newCleanOptions() = %+v, want all steps enabled", got)
	}
}

func TestNewCleanOptions_UsesRequestedSubset(t *testing.T) {
	got := newCleanOptions(true, false, true, false)
	if !got.projects || got.config || !got.managedResources || got.buildCache {
		t.Fatalf("newCleanOptions() = %+v, want only projects and managedResources enabled", got)
	}
}

func TestCleanProjectsSubsetSkipsEngineDetection(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	store, err := files.NewFileStore()
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(store.GetProjectBuildConfigPath("alpha")), 0755); err != nil {
		t.Fatalf("mkdir project dir: %v", err)
	}
	if err := os.WriteFile(store.GetProjectBuildConfigPath("alpha"), []byte("VERSION 1\n"), 0644); err != nil {
		t.Fatalf("write build.conf: %v", err)
	}
	if err := os.MkdirAll(store.GetGlobalDotfilesPath(), 0755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(store.GetGlobalDotfilesPath(), ".bashrc"), []byte("echo hi\n"), 0644); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	var out bytes.Buffer
	cons := console.New(context.Background(), strings.NewReader(""), &out, &out)

	if err := Clean(context.Background(), []string{"--no-prompt", "--projects"}, store, cons); err != nil {
		t.Fatalf("Clean(--projects) error = %v", err)
	}

	if _, err := os.Stat(store.GetProjectBuildConfigPath("alpha")); !os.IsNotExist(err) {
		t.Fatalf("expected project data to be removed, stat err = %v", err)
	}
	if _, err := os.Stat(filepath.Join(store.GetGlobalDotfilesPath(), ".bashrc")); err != nil {
		t.Fatalf("expected global config to be kept, stat err = %v", err)
	}
	if strings.Contains(out.String(), "Cleaning selected container engine") ||
		strings.Contains(out.String(), "Cleaning all available engines") {
		t.Fatalf("did not expect engine selection output, got:\n%s", out.String())
	}
}

func TestCleanHelpIncludesSubsetFlags(t *testing.T) {
	var out bytes.Buffer
	cons := console.New(context.Background(), strings.NewReader(""), &out, &out)

	if err := Clean(context.Background(), []string{"--help"}, nil, cons); err != nil {
		t.Fatalf("Clean(--help) error = %v", err)
	}

	got := out.String()
	for _, fragment := range []string{
		"Usage: paul-envs clean [flags]",
		"--engine string",
		"Container engine to clean: docker, podman, or all.",
		"--projects",
		"--config",
		"--managed-resources",
		"--build-cache",
	} {
		if !strings.Contains(got, fragment) {
			t.Fatalf("expected help output to contain %q, got:\n%s", fragment, got)
		}
	}
}
