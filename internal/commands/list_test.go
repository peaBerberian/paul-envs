package commands

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/peaberberian/paul-envs/internal/console"
	"github.com/peaberberian/paul-envs/internal/files"
)

func TestListNamesDoesNotEmitEngineMessages(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	store, err := files.NewFileStore()
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}

	projectName := "alpha"
	projectDir := filepath.Dir(store.GetProjectBuildConfigPath(projectName))
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("mkdir project dir: %v", err)
	}
	if err := os.WriteFile(store.GetProjectBuildConfigPath(projectName), []byte("VERSION 1\n"), 0644); err != nil {
		t.Fatalf("write build.conf: %v", err)
	}
	if err := os.WriteFile(store.GetProjectRuntimeConfigPath(projectName), []byte("PATH /tmp/alpha\n"), 0644); err != nil {
		t.Fatalf("write run.conf: %v", err)
	}

	var out bytes.Buffer
	cons := console.New(context.Background(), strings.NewReader(""), &out, &out)

	if err := List(context.Background(), []string{"--names"}, store, cons); err != nil {
		t.Fatalf("List(--names) error = %v", err)
	}

	if got := strings.TrimSpace(out.String()); got != projectName {
		t.Fatalf("List(--names) output = %q, want %q", got, projectName)
	}
}
