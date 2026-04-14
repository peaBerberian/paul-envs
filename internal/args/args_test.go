package args

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/peaberberian/paul-envs/internal/console"
	"github.com/peaberberian/paul-envs/internal/files"
)

func TestParseAndPrompt_NoPromptRejectsExactVersionWithoutMise(t *testing.T) {
	projectPath := t.TempDir()
	store, err := files.NewFileStore()
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}
	cons := console.New(context.Background(), strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})

	_, err = ParseAndPrompt([]string{
		projectPath,
		"--no-prompt",
		"--name", "test-no-mise-exact",
		"--nodejs", "20.10.0",
		"--no-mise",
	}, cons, store)
	if err == nil {
		t.Fatal("expected error for exact version without mise in --no-prompt mode, got nil")
	}
	if !strings.Contains(err.Error(), "exact language versions require Mise") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseAndPrompt_NoPromptAllowsLatestWithoutMise(t *testing.T) {
	projectPath := t.TempDir()
	store, err := files.NewFileStore()
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}
	cons := console.New(context.Background(), strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})

	cfg, err := ParseAndPrompt([]string{
		projectPath,
		"--no-prompt",
		"--name", "test-no-mise-latest",
		"--nodejs", "latest",
		"--no-mise",
	}, cons, store)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.InstallNode != "latest" {
		t.Fatalf("InstallNode = %q, want latest", cfg.InstallNode)
	}
}
