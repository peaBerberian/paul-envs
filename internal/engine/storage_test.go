package engine

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestStorageUsageByProject(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake engines require /bin/sh")
	}
	for _, binary := range []string{"docker", "podman"} {
		for _, scenario := range []string{"normal", "empty", "volume denied", "malformed volumes", "inspect denied", "malformed inspect", "missing container", "negative size", "list denied"} {
			t.Run(binary+"/"+scenario, func(t *testing.T) {
				dir := t.TempDir()
				logPath := filepath.Join(dir, "calls")
				installStorageTestEngine(t, binary, dir)
				t.Setenv("PAUL_STORAGE_CALLS", logPath)
				t.Setenv("PAUL_STORAGE_SCENARIO", scenario)
				var e ContainerEngine = &DockerEngine{}
				if binary == "podman" {
					e = &PodmanEngine{}
				}
				usage, err := e.GetStorageUsage(context.Background(), []string{"alpha", "beta", "gamma", "never-built"})
				wantError := strings.Contains(scenario, "denied") || strings.HasPrefix(scenario, "malformed")
				if (err != nil) != wantError {
					t.Fatalf("error = %v, want error %v", err, wantError)
				}
				if len(usage.Projects) != 4 {
					t.Fatalf("expected only requested project names: %+v", usage.Projects)
				}
				persisted, cache := int64(1024), int64(2048)
				if scenario == "empty" {
					persisted, cache = 0, 0
				}
				if scenario == "volume denied" || scenario == "malformed volumes" {
					assertStorageBytes(t, "shared cache", usage.SharedCacheBytes, nil)
					for project, size := range usage.Projects {
						assertStorageBytes(t, project+" persisted", size.PersistedBytes, nil)
					}
				} else {
					assertStorageBytes(t, "shared cache", usage.SharedCacheBytes, &cache)
					assertStorageBytes(t, "alpha persisted", usage.Projects["alpha"].PersistedBytes, &persisted)
					zero := int64(0)
					assertStorageBytes(t, "absent volume", usage.Projects["beta"].PersistedBytes, &zero)
					assertStorageBytes(t, "never built", usage.Projects["never-built"].PersistedBytes, &zero)
					var unknown *int64
					if scenario == "empty" {
						unknown = &zero
					}
					assertStorageBytes(t, "unmeasurable volume", usage.Projects["gamma"].PersistedBytes, unknown)
				}
				alpha, beta, zero := int64(3072), int64(4096), int64(0)
				wantAlpha, wantBeta, wantAbsent := &alpha, &beta, &zero
				switch scenario {
				case "empty":
					wantAlpha, wantBeta = &zero, &zero
				case "list denied":
					wantAlpha, wantBeta, wantAbsent = nil, nil, nil
				case "inspect denied", "malformed inspect":
					wantAlpha, wantBeta = nil, nil
				case "missing container", "negative size":
					wantAlpha = nil
				}
				assertStorageBytes(t, "alpha temporary (sum of two containers)", usage.Projects["alpha"].TemporaryBytes, wantAlpha)
				assertStorageBytes(t, "beta temporary", usage.Projects["beta"].TemporaryBytes, wantBeta)
				assertStorageBytes(t, "gamma temporary", usage.Projects["gamma"].TemporaryBytes, wantAbsent)
				assertStorageBytes(t, "never-built temporary", usage.Projects["never-built"].TemporaryBytes, wantAbsent)
				calls, err := os.ReadFile(logPath)
				if err != nil {
					t.Fatal(err)
				}
				if strings.Count(string(calls), "system\n") != 1 {
					t.Fatalf("volume usage must be queried once: %s", calls)
				}
			})
		}
	}
}

func installStorageTestEngine(t *testing.T, binary, dir string) {
	t.Helper()
	script := `#!/bin/sh
printf '%s\n' "$1" >> "$PAUL_STORAGE_CALLS"
case "$1" in
    system)
        [ "$2" = df ] && [ "$3" = --verbose ] || exit 9
        case "$0" in
            */docker) [ "$#" = 5 ] && [ "$4" = --format ] && [ "$5" = '{{json .Volumes}}' ] || exit 9 ;;
            */podman) [ "$#" = 3 ] || exit 9 ;;
        esac
        case "$PAUL_STORAGE_SCENARIO" in
            'volume denied') printf 'permission denied\n' >&2; exit 1 ;;
            'malformed volumes') printf 'incompatible report\n'; exit 0 ;;
        esac
        case "$0" in
            */docker)
                if [ "$PAUL_STORAGE_SCENARIO" = empty ]; then printf '[]\n'; exit 0; fi
                printf '%s\n' '[{"Size":"1.024kB","Name":"paulenv-alpha-local","Links":"2"},{"Name":"paulenv-gamma-local","Size":"N/A"},{"Name":"paulenv-shared-cache","Size":"2.048kB"},{"Name":"unrelated","Size":"999GB"}]'
                ;;
            */podman)
                printf 'Images space usage:\nignored\n\nLocal Volumes space usage:\n\nVOLUME NAME  LINKS  SIZE\n'
                if [ "$PAUL_STORAGE_SCENARIO" = empty ]; then exit 0; fi
                printf 'paulenv-alpha-local 2 1.024kB\npaulenv-gamma-local 0 N/A\npaulenv-shared-cache 2 2.048kB\nunrelated 0 999GB\n'
                ;;
        esac
        ;;
    ps)
        if [ "$PAUL_STORAGE_SCENARIO" = 'list denied' ]; then printf 'permission denied\n' >&2; exit 1; fi
        if [ "$PAUL_STORAGE_SCENARIO" = empty ]; then exit 0; fi
        printf 'aa\tpaulenv:alpha\tpaulenv-alpha\trunning\n'
        printf 'bb\tlocalhost/paulenv:alpha\tpaulenv-alpha-extra\texited\n'
        printf 'cc\timage-id\tpaulenv-beta\trunning\n'
        printf 'dd\tpaulenv:other\tpaulenv-other\trunning\n'
        printf 'ee\tunrelated\tunrelated\trunning\n'
        ;;
    container)
        # Unrequested and unrelated containers must not be inspected.
        [ "$*" = 'container inspect --size aa bb cc' ] || exit 9
        case "$PAUL_STORAGE_SCENARIO" in
            'inspect denied') printf 'permission denied\n' >&2; exit 1 ;;
            'malformed inspect') printf '[\n'; exit 0 ;;
            'missing container') printf '%s\n' '[{"Id":"aaaa","SizeRw":1024},{"Id":"cccc","SizeRw":4096}]' ;;
            'negative size') printf '%s\n' '[{"Id":"aaaa","SizeRw":1024},{"Id":"bbbb","SizeRw":-1},{"Id":"cccc","SizeRw":4096}]' ;;
            *) printf '%s\n' '[{"Id":"aaaa","SizeRw":1024,"SizeRootFs":999999},{"Id":"bbbb","SizeRw":2048},{"Id":"cccc","SizeRw":4096}]' ;;
        esac
        ;;
    *) exit 9 ;;
esac
`
	if err := os.WriteFile(filepath.Join(dir, binary), []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
}

func assertStorageBytes(t *testing.T, name string, got, want *int64) {
	t.Helper()
	if got == nil && want == nil {
		return
	}
	if got == nil || want == nil {
		t.Fatalf("%s: got %v, want %v", name, got, want)
	}
	if *got != *want {
		t.Fatalf("%s: got %d, want %d", name, *got, *want)
	}
}

func TestVolumeReportValidation(t *testing.T) {
	for _, report := range []string{`null`, `{}`, `[`, `[{}]`, `[{"Name":"volume","Size":123}]`} {
		if sizes, err := parseDockerVolumeSizes([]byte(report)); err == nil || sizes != nil {
			t.Fatalf("invalid Docker report must not become zero sizes: %q", report)
		}
	}
	for _, report := range []string{"", "Images space usage:\n", "Local Volumes space usage:\nvolume 1 0B", "Local Volumes space usage:\nVOLUME NAME LINKS SIZE\nvolume"} {
		if sizes, err := parsePodmanVolumeSizes(report); err == nil || sizes != nil {
			t.Fatalf("invalid Podman report must not become zero sizes: %q", report)
		}
	}
	for _, report := range []string{`[{"Name":"volume"}]`, `[{"Name":"volume","Size":null}]`} {
		sizes, err := parseDockerVolumeSizes([]byte(report))
		if value, ok := sizes["volume"]; err != nil || !ok || value != nil {
			t.Fatalf("missing size must remain unknown: %v %v", sizes, err)
		}
	}
}

func TestReportedSize(t *testing.T) {
	for _, tt := range []struct {
		value string
		bytes int64
	}{{"0B", 0}, {"1.024kB", 1024}, {"2.5 MB", 2500000}, {"1.5GiB", 1610612736}} {
		assertStorageBytes(t, tt.value, parseReportedSize(tt.value), &tt.bytes)
	}
	for _, value := range []string{"", "N/A", "-1B", "NaNB", "+InfB", "100000PB", "12unknown"} {
		assertStorageBytes(t, value, parseReportedSize(value), nil)
	}
}

func TestImageStorageMetadata(t *testing.T) {
	info := &ImageInfo{}
	if err := parseImageInfo([]byte(`{"Created":"2026-11-16T00:00:00Z","Size":1073741824}`), info); err != nil {
		t.Fatal(err)
	}
	bytes := int64(1 << 30)
	assertStorageBytes(t, "image", info.SizeBytes, &bytes)
	if info.BuiltAt == nil {
		t.Fatal("image timestamp lost")
	}
	for _, size := range []string{"null", "-1"} {
		if err := parseImageInfo([]byte(`{"Created":"2026-11-16T00:00:00Z","Size":`+size+`}`), info); err != nil {
			t.Fatal(err)
		}
		assertStorageBytes(t, "unknown image", info.SizeBytes, nil)
	}
	if err := parseImageInfo([]byte("malformed"), info); err == nil {
		t.Fatal("malformed image report must be reported")
	}
}
