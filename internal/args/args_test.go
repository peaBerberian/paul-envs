package args

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
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

func TestParseAndPrompt_NoPromptSeedDotfilesRequiresTemplate(t *testing.T) {
	projectPath := t.TempDir()
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	store, err := files.NewFileStore()
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}
	cons := console.New(context.Background(), strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})

	_, err = ParseAndPrompt([]string{
		projectPath,
		"--no-prompt",
		"--name", "test-seed-dotfiles",
		"--seed-dotfiles",
	}, cons, store)
	if err == nil {
		t.Fatal("expected error when --seed-dotfiles is used without a global template")
	}
	if !strings.Contains(err.Error(), "cannot seed dotfiles") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseAndPrompt_NoPromptSeedDotfilesUsesTemplate(t *testing.T) {
	projectPath := t.TempDir()
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	store, err := files.NewFileStore()
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}
	if err := os.MkdirAll(store.GetGlobalDotfilesPath(), 0755); err != nil {
		t.Fatalf("mkdir template dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(store.GetGlobalDotfilesPath(), ".bashrc"), []byte("echo hi\n"), 0644); err != nil {
		t.Fatalf("write template file: %v", err)
	}
	cons := console.New(context.Background(), strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})

	cfg, err := ParseAndPrompt([]string{
		projectPath,
		"--no-prompt",
		"--name", "test-seed-dotfiles",
		"--seed-dotfiles",
	}, cons, store)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.SeedDotfiles {
		t.Fatal("expected SeedDotfiles to be true")
	}
}
