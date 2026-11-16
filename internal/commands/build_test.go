package commands

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	versions "github.com/peaberberian/paul-envs/internal"
	"github.com/peaberberian/paul-envs/internal/console"
	"github.com/peaberberian/paul-envs/internal/files"
)

func TestBuildArguments(t *testing.T) {
	for _, tt := range []struct {
		name string
		args []string
		want string
	}{
		{"extra project", []string{"alpha", "beta"}, "at most one project name"},
		{"extra project with flags", []string{"alpha", "--no-cache", "beta"}, "at most one project name"},
		{"unknown flag", []string{"alpha", "--unknown"}, "flag provided but not defined"},
		{"missing value", []string{"alpha", "--engine"}, "flag needs an argument"},
		{"invalid value", []string{"alpha", "--no-cache=invalid"}, "invalid boolean value"},
		{"invalid engine", []string{"alpha", "--engine", "all"}, "invalid --engine value"},
		{"dash value", []string{"alpha", "--engine", "--"}, "invalid --engine value"},
		{"terminator", []string{"alpha", "--", "--no-cache"}, "at most one project name"},
		{"help after terminator", []string{"alpha", "--", "--help"}, "at most one project name"},
		{"long help before", []string{"--help", "alpha"}, ""},
		{"long help after", []string{"alpha", "--help"}, ""},
		{"short help before", []string{"-h", "alpha"}, ""},
		{"short help after", []string{"alpha", "-h"}, ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			ctx := context.Background()
			cons := console.New(ctx, strings.NewReader(""), &out, &out)
			err := Build(ctx, tt.args, nil, cons)
			if tt.want != "" {
				if err == nil || !strings.Contains(err.Error(), tt.want) {
					t.Fatalf("error = %v, want %q", err, tt.want)
				}
			} else if err != nil || !strings.Contains(out.String(), "Usage: paul-envs build") {
				t.Fatalf("help: error=%v output=%s", err, &out)
			}
		})
	}
}

func TestBuildFlagPlacement(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake engines require /bin/sh")
	}
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	store, err := files.NewFileStore()
	if err != nil {
		t.Fatal(err)
	}
	lockPath := filepath.Join(filepath.Dir(store.GetProjectBuildInfoPath("alpha")), "project.lock")
	for path, content := range map[string]string{
		store.GetProjectBuildConfigPath("alpha"):   "VERSION 1.0.0\nHOST_UID 1000\nHOST_GID 1000\nUSERNAME dev\nUSER_SHELL bash\n",
		store.GetProjectRuntimeConfigPath("alpha"): "VERSION 1.0.0\nPATH /tmp/alpha\n",
		lockPath: "VERSION=" + versions.ProjectLockVersion.ToString() + "\nDOCKERFILE_VERSION=" + versions.DockerfileVersion.ToString() + "\n",
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	dir := t.TempDir()
	logPath := filepath.Join(dir, "build-args")
	script := `#!/bin/sh
case "$1" in
    volume) printf 'paulenv-shared-cache\n' ;;
    build) printf '%s\n' "${0##*/}" "$@" > "$PAUL_ENVS_TEST_BUILD_ARGS" ;;
    --version)
        case "${0##*/}" in
            docker) printf 'Docker version 27.0.0\n' ;;
            podman) printf 'podman version 5.0.0\n' ;;
        esac
        ;;
    *) exit 1 ;;
esac
`
	for _, binary := range []string{"docker", "podman"} {
		if err := os.WriteFile(filepath.Join(dir, binary), []byte(script), 0700); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", dir)
	t.Setenv("PAUL_ENVS_TEST_BUILD_ARGS", logPath)
	for _, binary := range []string{"docker", "podman"} {
		for _, tt := range []struct {
			name    string
			args    []string
			noCache bool
		}{
			{"before", []string{"--engine", binary, "--no-cache", "alpha"}, true},
			{"after", []string{"alpha", "--no-cache", "--engine", binary}, true},
			{"mixed", []string{"--no-cache", "alpha", "--engine=" + binary}, true},
			{"terminator", []string{"--engine", binary, "--no-cache", "--", "alpha"}, true},
			{"default", []string{"alpha", "--engine", binary}, false},
			{"override", []string{"--no-cache", "alpha", "--no-cache=false", "--engine", binary}, false},
		} {
			t.Run(binary+"/"+tt.name, func(t *testing.T) {
				if err := os.WriteFile(logPath, nil, 0600); err != nil {
					t.Fatal(err)
				}
				var out bytes.Buffer
				ctx := context.Background()
				cons := console.New(ctx, strings.NewReader(""), &out, &out)
				if err := Build(ctx, tt.args, store, cons); err != nil {
					t.Fatalf("Build() error=%v output=%s", err, &out)
				}
				data, err := os.ReadFile(logPath)
				if err != nil {
					t.Fatal(err)
				}
				args := strings.Split(strings.TrimSpace(string(data)), "\n")
				if len(args) < 2 || args[0] != binary || args[1] != "build" {
					t.Fatalf("unexpected engine command: %q", args)
				}
				if slices.Contains(args, "--no-cache") != tt.noCache {
					t.Fatalf("engine args=%q, want no-cache=%v", args, tt.noCache)
				}
				if !slices.Contains(args, "paulenv:alpha") {
					t.Fatalf("wrong project image: %q", args)
				}
			})
		}
	}
}
