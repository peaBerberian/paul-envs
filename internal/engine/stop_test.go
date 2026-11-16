package engine

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"
)

func TestStopAndKillContainerCommands(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake engines require /bin/sh")
	}
	customTimeout, immediateTimeout, infiniteTimeout := 2, 0, -1
	for _, binary := range []string{"docker", "podman"} {
		for _, tt := range []struct {
			name    string
			action  string
			timeout *int
			fail    bool
		}{
			{name: "default timeout", action: "stop"},
			{name: "custom timeout", action: "stop", timeout: &customTimeout},
			{name: "immediate stop", action: "stop", timeout: &immediateTimeout},
			{name: "infinite timeout", action: "stop", timeout: &infiniteTimeout},
			{name: "kill", action: "kill"},
			{name: "stop failure", action: "stop", fail: true},
			{name: "kill failure", action: "kill", fail: true},
		} {
			t.Run(binary+"/"+tt.name, func(t *testing.T) {
				dir := t.TempDir()
				logPath := filepath.Join(dir, "args")
				script := `#!/bin/sh
[ "$1" = ps ] && exit 0
printf '%s\n' "$@" > "$PAUL_ENVS_TEST_ARGS"
[ "$PAUL_ENVS_TEST_FAILURE" != true ]
`
				if err := os.WriteFile(filepath.Join(dir, binary), []byte(script), 0700); err != nil {
					t.Fatal(err)
				}
				t.Setenv("PATH", dir)
				t.Setenv("PAUL_ENVS_TEST_ARGS", logPath)
				t.Setenv("PAUL_ENVS_TEST_FAILURE", "false")
				if tt.fail {
					t.Setenv("PAUL_ENVS_TEST_FAILURE", "true")
				}
				var containerEngine ContainerEngine = &DockerEngine{}
				timeoutFlag := "--timeout"
				if binary == "podman" {
					containerEngine = &PodmanEngine{}
					timeoutFlag = "--time"
				}
				container := ContainerInfo{ContainerId: "project-container-id"}
				var err error
				if tt.action == "stop" {
					err = containerEngine.StopContainer(context.Background(), container, StopOptions{Timeout: tt.timeout})
				} else {
					err = containerEngine.KillContainer(context.Background(), container)
				}
				if (err != nil) != tt.fail {
					t.Fatalf("error = %v, want failure %v", err, tt.fail)
				}
				data, err := os.ReadFile(logPath)
				if err != nil {
					t.Fatal(err)
				}
				want := []string{tt.action}
				if tt.timeout != nil {
					want = append(want, timeoutFlag, strconv.Itoa(*tt.timeout))
				}
				want = append(want, container.ContainerId)
				if got := strings.Split(strings.TrimSpace(string(data)), "\n"); !slices.Equal(got, want) {
					t.Fatalf("arguments = %q, want %q", got, want)
				}
			})
		}
	}
}
