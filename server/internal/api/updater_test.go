package api

import "testing"

func TestNormalizeVersion(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "plain", input: "0.1.0", want: "0.1.0"},
		{name: "prefixed lower", input: "v0.1.0", want: "0.1.0"},
		{name: "prefixed upper", input: "V1.2.3", want: "1.2.3"},
		{name: "trim spaces", input: "  v2.0.1  ", want: "2.0.1"},
		{name: "empty", input: "   ", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeVersion(tt.input); got != tt.want {
				t.Fatalf("normalizeVersion(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsNewerVersion(t *testing.T) {
	tests := []struct {
		name    string
		latest  string
		current string
		want    bool
	}{
		{name: "major upgrade", latest: "2.0.0", current: "1.9.9", want: true},
		{name: "minor upgrade", latest: "1.2.0", current: "1.1.9", want: true},
		{name: "patch upgrade", latest: "1.2.4", current: "1.2.3", want: true},
		{name: "same version", latest: "1.2.3", current: "1.2.3", want: false},
		{name: "current newer", latest: "1.2.3", current: "1.2.4", want: false},
		{name: "supports prefix", latest: "v1.2.3", current: "1.2.2", want: true},
		{name: "supports prerelease stripping", latest: "1.2.3-beta", current: "1.2.2", want: true},
		{name: "invalid latest", latest: "beta", current: "1.2.2", want: true},
		{name: "invalid equal fallback", latest: "beta", current: "beta", want: false},
		{name: "empty latest", latest: "", current: "1.2.2", want: false},
		{name: "empty current", latest: "1.2.2", current: "", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isNewerVersion(tt.latest, tt.current); got != tt.want {
				t.Fatalf("isNewerVersion(latest=%q, current=%q) = %v, want %v", tt.latest, tt.current, got, tt.want)
			}
		})
	}
}
