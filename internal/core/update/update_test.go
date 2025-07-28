package update

import (
	"testing"
)

func TestFunc_isVersionNewer(t *testing.T) {
	tests := []struct {
		current string
		latest  string
		want    bool
	}{
		{"1.0.0", "1.0.0", false},
		{"1.0.0", "1.0.1", true},
		{"1.0.1", "1.0.0", false},
		{"2.0.0", "1.9.9", false},
		{"2.2.3", "1.9.9", false},
		{"22.2.3", "1.9.9", false},
		{"1.2.3", "1.99.9", true},
		{"1.10", "1.5.99999", false},
	}

	for _, tt := range tests {
		t.Run(tt.current+" vs "+tt.latest, func(t *testing.T) {
			if got := isVersionNewer(Version(tt.current), Version(tt.latest)); got != tt.want {
				t.Errorf("isVersionNewer(%q, %q) = %v; want %v", tt.current, tt.latest, got, tt.want)
			}
		})
	}
}
