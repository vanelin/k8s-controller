package utils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExpandTilde(t *testing.T) {
	home, _ := os.UserHomeDir()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Expand tilde with path", "~/test", filepath.Join(home, "test")},
		{"Expand tilde only", "~", home},
		{"No tilde", "/tmp/file", "/tmp/file"},
		{"Empty string", "", ""},
		{"Relative path without tilde", "some/dir", "some/dir"},
		{"Tilde with subdirectory", "~/.kube/config", filepath.Join(home, ".kube/config")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExpandTilde(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
