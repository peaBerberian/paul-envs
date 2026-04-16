package engine

import (
	"path/filepath"
	"slices"
	"testing"

	"github.com/peaberberian/paul-envs/internal/files"
)

func TestDockerBuildArgs_NoCache(t *testing.T) {
	project := files.ProjectEntry{
		ProjectName:     "demo",
		BuildConfigPath: filepath.Join("/tmp", "paul-envs", "projects", "demo", "build.conf"),
	}

	args := dockerBuildArgs(project, map[string]string{"BETA": "2", "ALPHA": "1"}, BuildOptions{NoCache: true})

	if !slices.Contains(args, "--no-cache") {
		t.Fatalf("dockerBuildArgs() should include --no-cache, got %v", args)
	}
	if idxAlpha, idxBeta := slices.Index(args, "ALPHA=1"), slices.Index(args, "BETA=2"); idxAlpha == -1 || idxBeta == -1 || idxAlpha > idxBeta {
		t.Fatalf("dockerBuildArgs() should keep build args sorted, got %v", args)
	}
}

func TestPodmanBuildArgs_NoCache(t *testing.T) {
	project := files.ProjectEntry{
		ProjectName:     "demo",
		BuildConfigPath: filepath.Join("/tmp", "paul-envs", "projects", "demo", "build.conf"),
	}

	args := podmanBuildArgs(project, map[string]string{"BETA": "2", "ALPHA": "1"}, BuildOptions{NoCache: true})

	if !slices.Contains(args, "--no-cache") {
		t.Fatalf("podmanBuildArgs() should include --no-cache, got %v", args)
	}
	if idxAlpha, idxBeta := slices.Index(args, "ALPHA=1"), slices.Index(args, "BETA=2"); idxAlpha == -1 || idxBeta == -1 || idxAlpha > idxBeta {
		t.Fatalf("podmanBuildArgs() should keep build args sorted, got %v", args)
	}
}
