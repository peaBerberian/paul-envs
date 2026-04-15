package engine

import (
	"path/filepath"
	"testing"
)

func TestResolveRuntimePath(t *testing.T) {
	configPath := filepath.Join("/tmp", "projects", "demo", "run.conf")

	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "empty",
			in:   "",
			want: "",
		},
		{
			name: "absolute",
			in:   "/srv/dotfiles",
			want: "/srv/dotfiles",
		},
		{
			name: "relative",
			in:   "dotfiles",
			want: filepath.Join("/tmp", "projects", "demo", "dotfiles"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveRuntimePath(configPath, tt.in)
			if err != nil {
				t.Fatalf("resolveRuntimePath() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("resolveRuntimePath() = %q, want %q", got, tt.want)
			}
		})
	}
}
