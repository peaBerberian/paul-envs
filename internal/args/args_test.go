package args

import (
	"bytes"
	"context"
	"errors"
	"flag"
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

func TestParseFlags_CollectsRepeatableFlags(t *testing.T) {
	cons := console.New(context.Background(), strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})

	parsed, noPrompt, err := parseFlags([]string{
		"--no-prompt",
		"--package", "ripgrep",
		"--package", "fd-find",
		"--port", "3000",
		"--volume", "/tmp/cache:/cache:ro",
	}, cons)
	if err != nil {
		t.Fatalf("parseFlags() error = %v", err)
	}
	if !noPrompt {
		t.Fatal("expected noPrompt to be true")
	}
	if len(parsed.packages) != 2 || parsed.packages[0] != "ripgrep" || parsed.packages[1] != "fd-find" {
		t.Fatalf("unexpected packages: %#v", parsed.packages)
	}
	if len(parsed.ports) != 1 || parsed.ports[0] != "3000" {
		t.Fatalf("unexpected ports: %#v", parsed.ports)
	}
	if len(parsed.volumes) != 1 || parsed.volumes[0] != "/tmp/cache:/cache:ro" {
		t.Fatalf("unexpected volumes: %#v", parsed.volumes)
	}
}

func TestParseFlags_ReturnsHelpError(t *testing.T) {
	cons := console.New(context.Background(), strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})

	_, _, err := parseFlags([]string{"--help"}, cons)
	if !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("expected flag.ErrHelp, got %v", err)
	}
}

func TestPromptMissing_ConfirmsPrefilledLanguages(t *testing.T) {
	var out bytes.Buffer
	cfg := config.New("dev", config.ShellZsh)
	cfg.InstallNode = config.VersionLatest
	cfg.EnableSudo = true
	cfg.EnableSsh = true
	cfg.Packages = []string{"git"}
	cfg.Ports = []uint16{3000}
	cfg.Volumes = []string{"/tmp:/tmp"}

	cons := console.New(context.Background(), strings.NewReader("\n\n\n"), &out, &bytes.Buffer{})

	if err := promptMissing(cons, &cfg); err != nil {
		t.Fatalf("promptMissing() error = %v", err)
	}
	if cfg.InstallNode != config.VersionLatest {
		t.Fatalf("InstallNode = %q, want %q", cfg.InstallNode, config.VersionLatest)
	}
	output := out.String()
	if !strings.Contains(output, "Preselected language runtimes from CLI/config: Node.js (latest)") {
		t.Fatalf("missing language confirmation output:\n%s", output)
	}
	if strings.Contains(output, "Which language runtimes do you need?") {
		t.Fatalf("did not expect full language prompt after keeping selection:\n%s", output)
	}
}

func TestPromptMissing_RepromptsPrefilledDevTools(t *testing.T) {
	var out bytes.Buffer
	cfg := config.New("dev", config.ShellZsh)
	cfg.InstallNeovim = true
	cfg.EnableSudo = true
	cfg.EnableSsh = true
	cfg.Packages = []string{"git"}
	cfg.Ports = []uint16{3000}
	cfg.Volumes = []string{"/tmp:/tmp"}

	cons := console.New(context.Background(), strings.NewReader("\nn\n2 4\n\n"), &out, &bytes.Buffer{})

	if err := promptMissing(cons, &cfg); err != nil {
		t.Fatalf("promptMissing() error = %v", err)
	}
	if cfg.InstallNeovim {
		t.Fatal("expected Neovim to be cleared after reselecting tools")
	}
	if !cfg.InstallStarship || !cfg.InstallAtuin {
		t.Fatalf("expected Starship and Atuin to be selected, got starship=%v atuin=%v", cfg.InstallStarship, cfg.InstallAtuin)
	}
	output := out.String()
	if !strings.Contains(output, "Preselected development tools from CLI/config: Neovim") {
		t.Fatalf("missing tool confirmation output:\n%s", output)
	}
	if !strings.Contains(output, "Which of those tools do you want to install?") {
		t.Fatalf("expected full tool prompt after rejecting preselection:\n%s", output)
	}
}
