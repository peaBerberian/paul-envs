package files

import (
	"strings"
	"testing"
)

func TestGetCompletionScriptReturnsAssets(t *testing.T) {
	tests := []struct {
		shell       string
		wantPrefix  string
		wantContent string
	}{
		{shell: "bash", wantPrefix: "_paulenvs()", wantContent: "complete -F _paulenvs paul-envs"},
		{shell: "zsh", wantPrefix: "#compdef paul-envs", wantContent: "_arguments -C"},
		{shell: "fish", wantPrefix: "# Fish completion for paul-envs", wantContent: "complete -c paul-envs"},
	}

	for _, tt := range tests {
		got, err := GetCompletionScript(tt.shell)
		if err != nil {
			t.Fatalf("GetCompletionScript(%q) error = %v", tt.shell, err)
		}
		if !strings.HasPrefix(got, tt.wantPrefix) {
			t.Fatalf("GetCompletionScript(%q) prefix mismatch:\n%s", tt.shell, got)
		}
		if !strings.Contains(got, tt.wantContent) {
			t.Fatalf("GetCompletionScript(%q) missing %q", tt.shell, tt.wantContent)
		}
	}
}

func TestGetCompletionScriptRejectsUnknownShell(t *testing.T) {
	_, err := GetCompletionScript("tcsh")
	if err == nil {
		t.Fatal("GetCompletionScript(tcsh) error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "invalid shell") {
		t.Fatalf("unexpected error: %v", err)
	}
}
