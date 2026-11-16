package engine

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/peaberberian/paul-envs/internal/files"
)

func TestRunContainerEntrypointUser(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake container engine requires /bin/sh")
	}

	tests := []struct {
		name   string
		binary string
		engine ContainerEngine
		ci     string
		keepID bool
	}{
		{name: "podman keep-id", binary: "podman", engine: &PodmanEngine{}, keepID: true},
		{name: "podman CI fallback", binary: "podman", engine: &PodmanEngine{}, ci: "true"},
		{name: "docker image default", binary: "docker", engine: &DockerEngine{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.keepID && !supportsKeepID() {
				t.Skip("user namespaces are disabled")
			}
			dir := t.TempDir()
			argsPath := filepath.Join(dir, "args")
			project := files.ProjectEntry{
				ProjectName:       "demo",
				BuildConfigPath:   filepath.Join(dir, "build.conf"),
				RuntimeConfigPath: filepath.Join(dir, "run.conf"),
			}
			for path, content := range map[string]string{
				project.BuildConfigPath:   "VERSION 1.0.0\nHOST_UID 1000\nHOST_GID 1000\nUSERNAME dev\nUSER_SHELL bash\n",
				project.RuntimeConfigPath: "VERSION 1.0.0\nPATH /host/project\n",
			} {
				if err := os.WriteFile(path, []byte(content), 0600); err != nil {
					t.Fatal(err)
				}
			}
			script := `#!/bin/sh
case "$1" in
    volume)
        [ "$2" = ls ] || exit 1
        printf '%s\n' paulenv-shared-cache paulenv-demo-local
        ;;
    run) printf '%s\n' "$@" > "$PAUL_ENVS_TEST_ARGS" ;;
    *) exit 1 ;;
esac
`
			if err := os.WriteFile(filepath.Join(dir, tt.binary), []byte(script), 0700); err != nil {
				t.Fatal(err)
			}
			t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
			t.Setenv("PAUL_ENVS_TEST_ARGS", argsPath)
			t.Setenv("CI", tt.ci)

			command := []string{"id", "--user"}
			if err := tt.engine.RunContainer(context.Background(), project, command); err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(argsPath)
			if err != nil {
				t.Fatal(err)
			}
			args := strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
			imageIndex := slices.Index(args, "paulenv:demo")
			if imageIndex < 0 || !slices.Equal(args[imageIndex+1:], command) {
				t.Fatalf("image or command changed: %q", args)
			}
			options := args[:imageIndex]
			if got := slices.Contains(options, "--userns=keep-id"); got != tt.keepID {
				t.Fatalf("keep-id = %v, want %v: %q", got, tt.keepID, options)
			}
			userIndex := slices.Index(options, "--user")
			if tt.keepID {
				if userIndex < 0 || userIndex+1 >= len(options) || options[userIndex+1] != "0:0" {
					t.Fatalf("entrypoint must run as container root with keep-id: %q", options)
				}
			} else if userIndex >= 0 {
				t.Fatalf("must preserve the image's default user without keep-id: %q", options)
			}
		})
	}
}
