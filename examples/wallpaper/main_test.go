package main

import (
	"testing"
)

func TestShellSplit(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "simple command",
			input: "ls -la",
			want:  []string{"ls", "-la"},
		},
		{
			name:  "double-quoted argument",
			input: `feh --title "NASA APOD" %s`,
			want:  []string{"feh", "--title", "NASA APOD", "%s"},
		},
		{
			name:  "single-quoted argument",
			input: "feh --title 'NASA APOD' %s",
			want:  []string{"feh", "--title", "NASA APOD", "%s"},
		},
		{
			name:  "backslash escape",
			input: `echo hello\ world`,
			want:  []string{"echo", "hello world"},
		},
		{
			name:  "mixed quotes",
			input: `cmd 'a b' "c d" e`,
			want:  []string{"cmd", "a b", "c d", "e"},
		},
		{
			name:  "empty string",
			input: "",
			want:  nil,
		},
		{
			name:  "path with spaces",
			input: `feh "/tmp/My Image.jpg"`,
			want:  []string{"feh", "/tmp/My Image.jpg"},
		},
		{
			name:  "realistic feh command with spaces",
			input: `feh --title "NASA Wallpaper" --bg-scale "/tmp/My Image.jpg"`,
			want:  []string{"feh", "--title", "NASA Wallpaper", "--bg-scale", "/tmp/My Image.jpg"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shellSplit(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("shellSplit(%q) = %v (len %d), want %v (len %d)",
					tt.input, got, len(got), tt.want, len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("shellSplit(%q)[%d] = %q, want %q",
						tt.input, i, got[i], tt.want[i])
				}
			}
		})
	}
}
