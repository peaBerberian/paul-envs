package commands

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/peaberberian/paul-envs/internal/console"
	"github.com/peaberberian/paul-envs/internal/engine"
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

func TestCleanHelpIncludesEngineFlag(t *testing.T) {
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
	} {
		if !strings.Contains(got, fragment) {
			t.Fatalf("expected help output to contain %q, got:\n%s", fragment, got)
		}
	}
}
