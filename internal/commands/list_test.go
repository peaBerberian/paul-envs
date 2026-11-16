package commands

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/peaberberian/paul-envs/internal/console"
	"github.com/peaberberian/paul-envs/internal/engine"
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

func TestProjectRuntimeState(t *testing.T) {
	project, other := "alpha", "other"
	for _, tt := range []struct {
		name   string
		states []string
		want   string
	}{
		{name: "absent", want: "not running"},
		{name: "running", states: []string{"running"}, want: "running"},
		{name: "exited", states: []string{"exited"}, want: "stopped"},
		{name: "stopped", states: []string{"stopped"}, want: "stopped"},
		{name: "paused", states: []string{"paused"}, want: "paused"},
		{name: "created", states: []string{"created"}, want: "created"},
		{name: "configured", states: []string{"configured"}, want: "created"},
		{name: "initialized", states: []string{"initialized"}, want: "created"},
		{name: "restarting", states: []string{"restarting"}, want: "restarting"},
		{name: "removing", states: []string{"removing"}, want: "removing"},
		{name: "dead", states: []string{"dead"}, want: "dead"},
		{name: "unknown", states: []string{""}, want: "unknown"},
		{name: "multiple", states: []string{"exited", "running", "stopped"}, want: "running, stopped"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			containers := []engine.ContainerInfo{{State: "running"}, {ProjectName: &other, State: "running"}}
			for _, state := range tt.states {
				containers = append(containers, engine.ContainerInfo{ProjectName: &project, State: state})
			}
			if got := projectRuntimeState(project, containers); got != tt.want {
				t.Fatalf("runtime state = %q, want %q", got, tt.want)
			}
		})
	}
}

type listTestEngine struct {
	stubEngine
	image *engine.ImageInfo
}

func (s *listTestEngine) GetImageInfo(context.Context, string) (*engine.ImageInfo, error) {
	return s.image, nil
}

func TestListProjectInfoWithoutBuildMetadata(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	store, err := files.NewFileStore()
	if err != nil {
		t.Fatal(err)
	}
	builtAt := time.Now()
	image := &engine.ImageInfo{ImageName: "paulenv:alpha", BuiltAt: &builtAt}
	ctx := context.Background()
	cons := console.New(ctx, strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	cache := map[engine.Selection]*listEngineStatus{
		engine.SelectionPodman: {containerEngine: &listTestEngine{image: &engine.ImageInfo{ImageName: "localhost/paulenv:alpha"}}},
		engine.SelectionDocker: {containerEngine: &listTestEngine{image: image}},
	}
	info, err := listProjectInfo(ctx, "alpha", store, cons, cache)
	if err != nil || info.engineName != "docker" || info.image != image || info.runtimeState != "not running" {
		t.Fatalf("must find Docker image when Podman has no image: info=%+v err=%v", info, err)
	}

	cache[engine.SelectionDocker].err = errors.New("daemon unavailable")
	info, err = listProjectInfo(ctx, "alpha", store, cons, cache)
	if err == nil || info.runtimeState != "unknown" || info.image != image {
		t.Fatalf("must preserve image info but report runtime as unknown: info=%+v err=%v", info, err)
	}
}

func TestListRuntimeStatusWithEngines(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake engines require /bin/sh")
	}
	for _, binary := range []string{"docker", "podman"} {
		for _, failure := range []string{"", "daemon unavailable", "permission denied"} {
			t.Run(fmt.Sprintf("%s/failure=%s", binary, failure), func(t *testing.T) {
				t.Setenv("XDG_DATA_HOME", t.TempDir())
				t.Setenv("XDG_CONFIG_HOME", t.TempDir())
				store, err := files.NewFileStore()
				if err != nil {
					t.Fatal(err)
				}
				for _, name := range []string{"alpha", "beta", "gamma"} {
					for path, content := range map[string]string{
						store.GetProjectBuildConfigPath(name):   "VERSION 1.0.0\nHOST_UID 1000\nHOST_GID 1000\nUSERNAME dev\nUSER_SHELL bash\n",
						store.GetProjectRuntimeConfigPath(name): "VERSION 1.0.0\nPATH /tmp/" + name + "\n",
					} {
						if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
							t.Fatal(err)
						}
						if err := os.WriteFile(path, []byte(content), 0600); err != nil {
							t.Fatal(err)
						}
					}
					if err := os.MkdirAll(filepath.Dir(store.GetProjectBuildInfoPath(name)), 0755); err != nil {
						t.Fatal(err)
					}
					if err := store.RefreshBuildInfoFile(name, binary, "6.2.0"); err != nil {
						t.Fatal(err)
					}
				}
				dir := t.TempDir()
				logPath := filepath.Join(dir, "calls")
				script := `#!/bin/sh
printf '%s\n' "$1" >> "$PAUL_ENVS_TEST_CALLS"
case "$1" in
    ps)
        if [ -n "$PAUL_ENVS_TEST_FAILURE" ]; then
            printf '%s\n' "$PAUL_ENVS_TEST_FAILURE" >&2
            exit 1
        fi
        case "$*" in *'{{.State}}'*) ;; *) exit 1 ;; esac
        printf 'a\tpaulenv:alpha\tpaulenv-alpha\trunning\n'
        printf 'b\timage-id\tpaulenv-beta\texited\n'
        printf 'c\tunrelated:latest\tunrelated\trunning\n'
        ;;
    image)
        if [ -n "$PAUL_ENVS_TEST_FAILURE" ]; then
            printf '%s\n' "$PAUL_ENVS_TEST_FAILURE" >&2
            exit 1
        fi
        printf '%s\n' '2026-11-16T00:00:00Z'
        ;;
    *) exit 1 ;;
esac
`
				if err := os.WriteFile(filepath.Join(dir, binary), []byte(script), 0700); err != nil {
					t.Fatal(err)
				}
				t.Setenv("PATH", dir)
				t.Setenv("PAUL_ENVS_TEST_CALLS", logPath)
				t.Setenv("PAUL_ENVS_TEST_FAILURE", failure)
				var out bytes.Buffer
				cons := console.New(context.Background(), strings.NewReader(""), &out, &out)
				if err := List(context.Background(), nil, store, cons); err != nil {
					t.Fatal(err)
				}
				if got := strings.Count(out.String(), "("+binary+")"); failure == "" && got != 3 {
					t.Fatalf("expected engine on each project, got:\n%s", &out)
				}
				if failure != "" {
					if strings.Count(out.String(), "Status            : unknown") != 3 || !strings.Contains(out.String(), failure) {
						t.Fatalf("failed queries must show unknown and a warning:\n%s", &out)
					}
					if strings.Count(out.String(), failure) != 3 {
						t.Fatalf("the same failure should only appear once per project:\n%s", &out)
					}
				} else {
					for _, state := range []string{"running", "stopped", "not running"} {
						if !strings.Contains(out.String(), "Status            : "+state) {
							t.Fatalf("missing state %q:\n%s", state, &out)
						}
					}
					calls, err := os.ReadFile(logPath)
					if err != nil {
						t.Fatal(err)
					}
					if strings.Count(string(calls), "ps\n") != 1 {
						t.Fatalf("container list should be queried once per engine: %s", calls)
					}
				}
			})
		}
	}
}
