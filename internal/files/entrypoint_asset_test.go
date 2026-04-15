package files

import (
	"strings"
	"testing"
)

func TestEmbeddedEntrypointContainsRuntimeManagement(t *testing.T) {
	content, err := assets.ReadFile("embeds/entrypoint.sh")
	if err != nil {
		t.Fatalf("read embedded entrypoint: %v", err)
	}
	script := string(content)

	checks := []string{
		"write_shell_overrides()",
		"sync_dotfiles()",
		"apply_git_config()",
		"--exclude=.container-cache",
		"--exclude=.container-overrides.bash",
		"paul-envs managed bash overrides",
		"paul-envs managed zsh overrides",
		"paul-envs managed fish overrides",
		"git config --global user.name",
		"git config --global user.email",
	}
	for _, check := range checks {
		if !strings.Contains(script, check) {
			t.Fatalf("entrypoint missing expected content: %s", check)
		}
	}
}
