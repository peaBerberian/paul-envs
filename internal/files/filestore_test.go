package files

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestNewFileStore(t *testing.T) {
	store, err := NewFileStore()
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}
	if store == nil {
		t.Fatal("NewFileStore() returned nil")
	}
	if store.baseDataDir == "" {
		t.Error("baseDataDir should not be empty")
	}
	if store.baseConfigDir == "" {
		t.Error("baseConfigDir should not be empty")
	}
	if store.projectsDir == "" {
		t.Error("projectsDir should not be empty")
	}
}

func TestFileStore_PathHelpers(t *testing.T) {
	store := &FileStore{
		baseDataDir:   "/test/base",
		baseConfigDir: "/test/config",
		projectsDir:   "/test/base/projects",
	}

	tests := []struct {
		name     string
		fn       func() string
		expected string
	}{
		{
			name:     "getProjectDirBase",
			fn:       store.getProjectDirBase,
			expected: "/test/base/projects",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.fn()
			if got != tt.expected {
				t.Errorf("%s = %v, want %v", tt.name, got, tt.expected)
			}
		})
	}
}

func TestFileStore_GetProjectDir(t *testing.T) {
	store := &FileStore{
		baseDataDir:   "/test/base",
		baseConfigDir: "/test/config",
		projectsDir:   "/test/base/projects",
	}

	got := store.getProjectDir("myproject")
	expected := "/test/base/projects/myproject"
	if got != expected {
		t.Errorf("GetProjectDir() = %v, want %v", got, expected)
	}
}

func TestFileStore_GetProjectInternalDir(t *testing.T) {
	store := &FileStore{
		baseDataDir:   "/test/base",
		baseConfigDir: "/test/config",
		projectsDir:   "/test/base/projects",
	}

	got := store.getProjectInternalDir("myproject")
	expected := "/test/base/projects/myproject/.paul-env"
	if got != expected {
		t.Errorf("getProjectInternalDir() = %v, want %v", got, expected)
	}
}

func TestFileStore_GetProjectBuildConfigPath(t *testing.T) {
	store := &FileStore{
		baseDataDir:   "/test/base",
		baseConfigDir: "/test/config",
		projectsDir:   "/test/base/projects",
	}

	got := store.GetProjectBuildConfigPath("myproject")
	expected := "/test/base/projects/myproject/build.conf"
	if got != expected {
		t.Errorf("GetProjectBuildConfigPath() = %v, want %v", got, expected)
	}
}

func TestFileStore_GetProjectRuntimeConfigPath(t *testing.T) {
	store := &FileStore{
		baseDataDir:   "/test/base",
		baseConfigDir: "/test/config",
		projectsDir:   "/test/base/projects",
	}

	got := store.GetProjectRuntimeConfigPath("myproject")
	expected := "/test/base/projects/myproject/run.conf"
	if got != expected {
		t.Errorf("GetProjectRuntimeConfigPath() = %v, want %v", got, expected)
	}
}

func TestFileStore_GetProjectDotfilesPath(t *testing.T) {
	store := &FileStore{
		baseDataDir:   "/test/base",
		baseConfigDir: "/test/config",
		projectsDir:   "/test/base/projects",
	}

	got := store.GetProjectDotfilesPath("myproject")
	expected := "/test/base/projects/myproject/dotfiles"
	if got != expected {
		t.Errorf("GetProjectDotfilesPath() = %v, want %v", got, expected)
	}
}

func TestFileStore_GetGlobalDotfilesPath(t *testing.T) {
	store := &FileStore{
		baseDataDir:   "/test/base",
		baseConfigDir: "/test/config",
		projectsDir:   "/test/base/projects",
	}

	got := store.GetGlobalDotfilesPath()
	expected := "/test/config/dotfiles"
	if got != expected {
		t.Errorf("GetGlobalDotfilesPath() = %v, want %v", got, expected)
	}
}

func TestFileStore_HasGlobalDotfilesTemplate(t *testing.T) {
	baseConfigDir := t.TempDir()
	store := &FileStore{
		userFS:        &UserFS{homeDir: t.TempDir(), sudoUser: nil},
		baseDataDir:   t.TempDir(),
		baseConfigDir: baseConfigDir,
		projectsDir:   filepath.Join(t.TempDir(), "projects"),
	}

	hasTemplate, err := store.HasGlobalDotfilesTemplate()
	if err != nil {
		t.Fatalf("HasGlobalDotfilesTemplate() error = %v", err)
	}
	if hasTemplate {
		t.Fatal("expected no template when directory is missing")
	}

	dotfilesDir := store.GetGlobalDotfilesPath()
	if err := os.MkdirAll(dotfilesDir, 0755); err != nil {
		t.Fatalf("mkdir global dotfiles: %v", err)
	}
	hasTemplate, err = store.HasGlobalDotfilesTemplate()
	if err != nil {
		t.Fatalf("HasGlobalDotfilesTemplate() error = %v", err)
	}
	if hasTemplate {
		t.Fatal("expected empty directory to not count as template")
	}

	if err := os.WriteFile(filepath.Join(dotfilesDir, ".bashrc"), []byte("echo hi\n"), 0644); err != nil {
		t.Fatalf("write template file: %v", err)
	}
	hasTemplate, err = store.HasGlobalDotfilesTemplate()
	if err != nil {
		t.Fatalf("HasGlobalDotfilesTemplate() error = %v", err)
	}
	if !hasTemplate {
		t.Fatal("expected non-empty directory to count as template")
	}
}

func TestFileStore_SeedProjectDotfiles(t *testing.T) {
	baseDataDir := t.TempDir()
	baseConfigDir := t.TempDir()
	store := &FileStore{
		userFS:        &UserFS{homeDir: t.TempDir(), sudoUser: nil},
		baseDataDir:   baseDataDir,
		baseConfigDir: baseConfigDir,
		projectsDir:   filepath.Join(baseDataDir, "projects"),
	}

	projectName := "myproject"
	if err := os.MkdirAll(store.GetProjectDotfilesPath(projectName), 0755); err != nil {
		t.Fatalf("mkdir project dotfiles: %v", err)
	}
	if err := os.MkdirAll(store.GetGlobalDotfilesPath(), 0755); err != nil {
		t.Fatalf("mkdir global dotfiles: %v", err)
	}
	if err := os.WriteFile(filepath.Join(store.GetGlobalDotfilesPath(), ".gitconfig"), []byte("[user]\n"), 0644); err != nil {
		t.Fatalf("write template file: %v", err)
	}

	if err := store.SeedProjectDotfiles(context.Background(), projectName); err != nil {
		t.Fatalf("SeedProjectDotfiles() error = %v", err)
	}

	if _, err := os.Stat(filepath.Join(store.GetProjectDotfilesPath(projectName), ".gitconfig")); err != nil {
		t.Fatalf("expected seeded file to exist: %v", err)
	}
}
