package utils

import (
	"os"
	"path/filepath"
)

// ExpandTilde expands the tilde (~) to the user's home directory
func ExpandTilde(path string) string {
	if len(path) > 0 && path[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return path // Return original path if we can't get home directory
		}
		return filepath.Join(home, path[1:])
	}
	return path
}
