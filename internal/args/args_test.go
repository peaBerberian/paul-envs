package args

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/peaberberian/paul-envs/internal/config"
	"github.com/peaberberian/paul-envs/internal/console"
	"github.com/peaberberian/paul-envs/internal/files"
)

func TestBuildConfig_LeavesLanguagesUnsetForInteractivePrompting(t *testing.T) {
	cfg, err := buildConfig(t.TempDir(), &parsedFlags{})
	if err != nil {
		t.Fatalf("buildConfig() error = %v", err)
	}
	if cfg.InstallNode != "" || cfg.InstallRust != "" || cfg.InstallPython != "" || cfg.InstallGo != "" {
		t.Fatalf("expected unset language versions before prompting, got node=%q rust=%q python=%q go=%q",
			cfg.InstallNode, cfg.InstallRust, cfg.InstallPython, cfg.InstallGo)
	}
}

func TestValidateNoPromptConfig_DefaultsUnsetLanguagesToNone(t *testing.T) {
	cfg := config.New("dev", config.ShellBash)

	if err := validateNoPromptConfig(&cfg); err != nil {
		t.Fatalf("validateNoPromptConfig() error = %v", err)
	}
	if cfg.InstallNode != config.VersionNone ||
		cfg.InstallRust != config.VersionNone ||
		cfg.InstallPython != config.VersionNone ||
		cfg.InstallGo != config.VersionNone {
		t.Fatalf("expected no-prompt defaults to be %q, got node=%q rust=%q python=%q go=%q",
			config.VersionNone, cfg.InstallNode, cfg.InstallRust, cfg.InstallPython, cfg.InstallGo)
	}
}

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
