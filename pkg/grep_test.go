package pkg

import (
	"fmt"
	"testing"
)

func TestMatch(t *testing.T) {
	tests := []struct {
		pattern string
		s       string
		want    bool
	}{
		{"*a", "a", true},
		{"*a", "ba", true},
		{"*a", "bb", false},
		{"*a", "ab", false},
		{"a*", "a", true},
		{"a*", "ab", true},
		{"a*", "bb", false},
		{"a*", "ba", false},
		{"*", "ba", true},
		{"*", "", true},
		{"*", "a", true},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s-%s", tt.pattern, tt.s), func(t *testing.T) {
			if got := match(tt.pattern, tt.s); got != tt.want {
				t.Errorf("match() = %v, want %v", got, tt.want)
			}
		})
	}
}
