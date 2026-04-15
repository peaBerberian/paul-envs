package config

import (
	"os"
	"path/filepath"
	"testing"
)

// writeConf writes content to a temp file and returns its path.
// The file is automatically removed when the test ends.
func writeConf(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "*.conf")
	if err != nil {
		t.Fatalf("could not create temp file: %v", err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatalf("could not write temp file: %v", err)
	}
	f.Close()
	return f.Name()
}

func TestLoadRuntimeConfig_Minimal(t *testing.T) {
	path := writeConf(t, "VERSION 1.0.0\nPATH /srv/myproject\n")
	cfg, err := LoadRuntimeConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ProjectPath != "/srv/myproject" {
		t.Errorf("ProjectPath: want /srv/myproject, got %q", cfg.ProjectPath)
	}
	if len(cfg.Volumes) != 0 {
		t.Errorf("Volumes: want empty, got %v", cfg.Volumes)
	}
	if len(cfg.Ports) != 0 {
		t.Errorf("Ports: want empty, got %v", cfg.Ports)
	}
	if cfg.WorkDir != "" {
		t.Errorf("WorkDir: want empty, got %q", cfg.WorkDir)
	}
	if cfg.DotfilesPath != "" {
		t.Errorf("DotfilesPath: want empty, got %q", cfg.DotfilesPath)
	}
	if cfg.GitName != "" {
		t.Errorf("GitName: want empty, got %q", cfg.GitName)
	}
	if cfg.GitEmail != "" {
		t.Errorf("GitEmail: want empty, got %q", cfg.GitEmail)
	}
}

func TestLoadRuntimeConfig_AllDirectives(t *testing.T) {
	content := `PATH /srv/myproject
VERSION 1.0.0
VOLUME /host/data:/container/data
VOLUME /host/logs:/container/logs
PORT 8080:80
PORT 5432:5432
WORKDIR /container/data
DOTFILES_PATH ./dotfiles
GIT_AUTHOR_NAME Jane Doe
GIT_AUTHOR_EMAIL jane@example.com
`
	cfg, err := LoadRuntimeConfig(writeConf(t, content))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.ProjectPath != "/srv/myproject" {
		t.Errorf("ProjectPath: want /srv/myproject, got %q", cfg.ProjectPath)
	}

	wantVolumes := []string{"/host/data:/container/data", "/host/logs:/container/logs"}
	assertStringSlice(t, "Volumes", wantVolumes, cfg.Volumes)

	wantPorts := []string{"8080:80", "5432:5432"}
	assertStringSlice(t, "Ports", wantPorts, cfg.Ports)

	if cfg.WorkDir != "/container/data" {
		t.Errorf("WorkDir: want /container/data, got %q", cfg.WorkDir)
	}
	if cfg.DotfilesPath != "./dotfiles" {
		t.Errorf("DotfilesPath: want ./dotfiles, got %q", cfg.DotfilesPath)
	}
	if cfg.GitName != "Jane Doe" {
		t.Errorf("GitName: want Jane Doe, got %q", cfg.GitName)
	}
	if cfg.GitEmail != "jane@example.com" {
		t.Errorf("GitEmail: want jane@example.com, got %q", cfg.GitEmail)
	}
}

func TestLoadRuntimeConfig_CommentsAndBlanksIgnored(t *testing.T) {
	content := `
VERSION 1.0.0
# project root
PATH /srv/myproject

# extra storage
VOLUME /host/data:/container/data
`
	cfg, err := LoadRuntimeConfig(writeConf(t, content))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ProjectPath != "/srv/myproject" {
		t.Errorf("ProjectPath: want /srv/myproject, got %q", cfg.ProjectPath)
	}
	assertStringSlice(t, "Volumes", []string{"/host/data:/container/data"}, cfg.Volumes)
}

func TestLoadRuntimeConfig_TildeInVolume(t *testing.T) {
	// ~/ in a VOLUME value should be expanded by the underlying parser.
	// We only assert that the result does NOT start with '~'.
	content := "PATH /srv/myproject\nVOLUME ~/data:/container/data\n"
	content = "VERSION 1.0.0\n" + content
	cfg, err := LoadRuntimeConfig(writeConf(t, content))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Volumes) != 1 {
		t.Fatalf("Volumes: want 1, got %d", len(cfg.Volumes))
	}
	if cfg.Volumes[0][:1] == "~" {
		t.Errorf("tilde was not expanded in VOLUME value: %q", cfg.Volumes[0])
	}
}

func TestLoadRuntimeConfig_LastPathWins(t *testing.T) {
	// If PATH appears more than once the last value should win, consistent with
	// how most config formats behave when a scalar directive is repeated.
	content := "PATH /first\nPATH /second\n"
	content = "VERSION 1.0.0\n" + content
	cfg, err := LoadRuntimeConfig(writeConf(t, content))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ProjectPath != "/second" {
		t.Errorf("ProjectPath: want /second, got %q", cfg.ProjectPath)
	}
}

func TestLoadRuntimeConfig_LastWorkdirWins(t *testing.T) {
	content := "PATH /srv/myproject\nWORKDIR /first\nWORKDIR /second\n"
	content = "VERSION 1.0.0\n" + content
	cfg, err := LoadRuntimeConfig(writeConf(t, content))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.WorkDir != "/second" {
		t.Errorf("WorkDir: want /second, got %q", cfg.WorkDir)
	}
}

func TestLoadRuntimeConfig_LastRuntimeIdentityWins(t *testing.T) {
	content := "PATH /srv/myproject\nDOTFILES_PATH ./one\nDOTFILES_PATH ./two\nGIT_AUTHOR_NAME One\nGIT_AUTHOR_NAME Two\nGIT_AUTHOR_EMAIL one@example.com\nGIT_AUTHOR_EMAIL two@example.com\n"
	content = "VERSION 1.0.0\n" + content
	cfg, err := LoadRuntimeConfig(writeConf(t, content))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.DotfilesPath != "./two" {
		t.Errorf("DotfilesPath: want ./two, got %q", cfg.DotfilesPath)
	}
	if cfg.GitName != "Two" {
		t.Errorf("GitName: want Two, got %q", cfg.GitName)
	}
	if cfg.GitEmail != "two@example.com" {
		t.Errorf("GitEmail: want two@example.com, got %q", cfg.GitEmail)
	}
}

func TestLoadRuntimeConfig_MissingPath(t *testing.T) {
	content := "VOLUME /host/data:/container/data\nPORT 8080:80\n"
	content = "VERSION 1.0.0\n" + content
	_, err := LoadRuntimeConfig(writeConf(t, content))
	if err == nil {
		t.Fatal("expected error for missing PATH, got nil")
	}
}

func TestLoadRuntimeConfig_EmptyFile(t *testing.T) {
	_, err := LoadRuntimeConfig(writeConf(t, ""))
	if err == nil {
		t.Fatal("expected error for empty file (missing PATH), got nil")
	}
}

func TestLoadRuntimeConfig_UnknownDirective(t *testing.T) {
	content := "VERSION 1.0.0\nPATH /srv/myproject\nFOO bar\n"
	_, err := LoadRuntimeConfig(writeConf(t, content))
	if err == nil {
		t.Fatal("expected error for unknown directive, got nil")
	}
}

func TestLoadRuntimeConfig_MissingVersion(t *testing.T) {
	_, err := LoadRuntimeConfig(writeConf(t, "PATH /srv/myproject\n"))
	if err == nil {
		t.Fatal("expected error for missing VERSION, got nil")
	}
}

func TestLoadRuntimeConfig_FileNotFound(t *testing.T) {
	_, err := LoadRuntimeConfig(filepath.Join(t.TempDir(), "nonexistent.conf"))
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func assertStringSlice(t *testing.T, name string, want, got []string) {
	t.Helper()
	if len(want) != len(got) {
		t.Fatalf("%s: length mismatch: want %d, got %d\nwant: %v\ngot:  %v", name, len(want), len(got), want, got)
	}
	for i := range want {
		if want[i] != got[i] {
			t.Errorf("%s[%d]: want %q, got %q", name, i, want[i], got[i])
		}
	}
}
