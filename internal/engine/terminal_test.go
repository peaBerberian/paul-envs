package engine

import (
	"reflect"
	"testing"
)

func TestContainerTerminalEnvArgs(t *testing.T) {
	tests := []struct {
		name      string
		term      string
		colorTerm string
		want      []string
	}{
		{
			name:      "truecolor terminal",
			term:      "foot",
			colorTerm: "truecolor",
			want:      []string{"--env", "TERM=xterm-256color", "--env", "COLORTERM=truecolor"},
		},
		{
			name:      "24 bit terminal",
			term:      "xterm",
			colorTerm: "24bit",
			want:      []string{"--env", "TERM=xterm-256color", "--env", "COLORTERM=24bit"},
		},
		{
			name: "256 color terminal",
			term: "tmux-256color",
			want: []string{"--env", "TERM=xterm-256color"},
		},
		{
			name: "direct color terminfo",
			term: "xterm-direct",
			want: []string{"--env", "TERM=xterm-256color"},
		},
		{
			name: "basic terminal",
			term: "xterm",
			want: []string{"--env", "TERM=xterm"},
		},
		{
			name:      "dumb terminal stays dumb",
			term:      "dumb",
			colorTerm: "truecolor",
			want:      []string{"--env", "TERM=dumb"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := containerTerminalEnvArgs(tt.term, tt.colorTerm)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("containerTerminalEnvArgs(%q, %q) = %q, want %q", tt.term, tt.colorTerm, got, tt.want)
			}
		})
	}
}
