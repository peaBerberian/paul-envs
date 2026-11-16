package commands

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/peaberberian/paul-envs/internal/console"
	"github.com/peaberberian/paul-envs/internal/files"
)

func TestStopAndKillArguments(t *testing.T) {
	ctx := context.Background()
	for _, action := range []string{"stop", "kill"} {
		for _, tt := range []struct {
			name string
			args []string
			want string
		}{
			{name: "missing project", want: "requires exactly one project name"},
			{name: "extra project", args: []string{"alpha", "beta"}, want: "requires exactly one project name"},
			{name: "invalid project", args: []string{"../alpha"}, want: "invalid project name"},
			{name: "invalid engine", args: []string{"--engine", "all", "alpha"}, want: "invalid --engine value"},
			{name: "invalid engine after project", args: []string{"alpha", "--engine", "all"}, want: "invalid --engine value"},
			{name: "unknown flag", args: []string{"--unknown"}, want: "flag provided but not defined"},
			{name: "long help", args: []string{"--help"}},
			{name: "short help", args: []string{"-h"}},
			{name: "help after project", args: []string{"alpha", "--help"}},
		} {
			t.Run(action+"/"+tt.name, func(t *testing.T) {
				var out bytes.Buffer
				cons := console.New(ctx, strings.NewReader(""), &out, &out)
				err := terminateProject(ctx, tt.args, nil, cons, action)
				if tt.want != "" {
					if err == nil || !strings.Contains(err.Error(), tt.want) {
						t.Fatalf("error = %v, want %q", err, tt.want)
					}
				} else if err != nil || !strings.Contains(out.String(), "Usage: paul-envs "+action) || !strings.Contains(out.String(), "--engine") {
					t.Fatalf("help: error=%v output=%s", err, &out)
				}
			})
		}
	}
	for _, args := range [][]string{{"--timeout", "-2", "alpha"}, {"--timeout", "invalid", "alpha"}} {
		cons := console.New(ctx, strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
		if err := Stop(ctx, args, nil, cons); err == nil {
			t.Fatalf("accepted invalid timeout: %q", args)
		}
	}
	cons := console.New(ctx, strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	if err := Kill(ctx, []string{"--timeout", "1", "alpha"}, nil, cons); err == nil {
		t.Fatal("kill accepted a stop-only flag")
	}
}

func TestStopAndKillProjectSelection(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake engines require /bin/sh")
	}
	for _, binary := range []string{"docker", "podman"} {
		for _, action := range []string{"stop", "kill"} {
			for _, scenario := range []string{"recorded engine", "explicit engine", "inactive", "missing project", "unknown state", "list failure", "action failure"} {
				t.Run(binary+"/"+action+"/"+scenario, func(t *testing.T) {
					t.Setenv("XDG_DATA_HOME", t.TempDir())
					t.Setenv("XDG_CONFIG_HOME", t.TempDir())
					store, err := files.NewFileStore()
					if err != nil {
						t.Fatal(err)
					}
					buildPath := store.GetProjectBuildConfigPath("alpha")
					runPath := store.GetProjectRuntimeConfigPath("alpha")
					for path, content := range map[string]string{
						buildPath: "VERSION 1.0.0\nHOST_UID 1000\nHOST_GID 1000\nUSERNAME dev\nUSER_SHELL bash\n",
						runPath:   "VERSION 1.0.0\nPATH /tmp/alpha\n",
					} {
						if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
							t.Fatal(err)
						}
						if err := os.WriteFile(path, []byte(content), 0600); err != nil {
							t.Fatal(err)
						}
					}
					if err := os.MkdirAll(filepath.Dir(store.GetProjectBuildInfoPath("alpha")), 0755); err != nil {
						t.Fatal(err)
					}
					recorded := binary
					args := []string{}
					if scenario == "explicit engine" {
						if binary == "docker" {
							recorded = "podman"
						} else {
							recorded = "docker"
						}
						args = append(args, "--engine", binary)
					}
					if err := store.RefreshBuildInfoFile("alpha", recorded, "6.2.0"); err != nil {
						t.Fatal(err)
					}
					if action == "stop" {
						args = append(args, "--timeout", "2")
					}
					name := "alpha"
					if scenario == "missing project" {
						name = "missing"
					}
					args = append(args, name)
					dir := t.TempDir()
					logPath := filepath.Join(dir, "calls")
					script := `#!/bin/sh
case "$1" in
    ps)
        [ "$PAUL_ENVS_TEST_SCENARIO" != 'list failure' ] || exit 1
        printf 'unrelated\tpaulenv:other\tpaulenv-other\trunning\n'
        printf 'stopped\tpaulenv:alpha\tpaulenv-alpha-stopped\texited\n'
        printf 'created\tpaulenv:alpha\tpaulenv-alpha-created\tcreated\n'
        case "$PAUL_ENVS_TEST_SCENARIO" in
            inactive) ;;
            'unknown state') printf 'unknown\tpaulenv:alpha\tpaulenv-alpha\tunknown\n' ;;
            *)
                printf 'active-1\tpaulenv:alpha\tpaulenv-alpha\trunning\n'
                printf 'active-2\tpaulenv:alpha\tpaulenv-alpha-second\trestarting\n'
                ;;
        esac
        ;;
    stop|kill)
        printf '%s\n' "$*" >> "$PAUL_ENVS_TEST_CALLS"
        [ "$PAUL_ENVS_TEST_SCENARIO" != 'action failure' ]
        ;;
    *) exit 1 ;;
esac
`
					if err := os.WriteFile(filepath.Join(dir, binary), []byte(script), 0700); err != nil {
						t.Fatal(err)
					}
					t.Setenv("PATH", dir)
					t.Setenv("PAUL_ENVS_TEST_CALLS", logPath)
					t.Setenv("PAUL_ENVS_TEST_SCENARIO", scenario)
					var out bytes.Buffer
					ctx := context.Background()
					cons := console.New(ctx, strings.NewReader(""), &out, &out)
					if action == "stop" {
						err = Stop(ctx, args, store, cons)
					} else {
						err = Kill(ctx, args, store, cons)
					}
					wantFailure := scenario == "missing project" || scenario == "unknown state" || strings.HasSuffix(scenario, "failure")
					if (err != nil) != wantFailure {
						t.Fatalf("error=%v want failure=%v output=%s", err, wantFailure, &out)
					}
					calls, readErr := os.ReadFile(logPath)
					wantCalls := ""
					if scenario == "recorded engine" || scenario == "explicit engine" || scenario == "action failure" {
						prefix := action
						if action == "stop" {
							if binary == "docker" {
								prefix += " --timeout 2"
							} else {
								prefix += " --time 2"
							}
						}
						wantCalls = prefix + " active-1\n"
						if scenario != "action failure" {
							wantCalls += prefix + " active-2\n"
						}
					}
					if readErr != nil && !os.IsNotExist(readErr) {
						t.Fatal(readErr)
					}
					if string(calls) != wantCalls {
						t.Fatalf("calls=%q want=%q", calls, wantCalls)
					}
					if scenario == "inactive" && !strings.Contains(out.String(), "No active containers") {
						t.Fatalf("missing inactive status: %s", &out)
					}
					for _, path := range []string{buildPath, runPath, store.GetProjectBuildInfoPath("alpha")} {
						if _, err := os.Stat(path); err != nil {
							t.Fatalf("project metadata was removed: %v", err)
						}
					}
				})
			}
		}
	}
}
