package utils

import (
	"strings"
)

// IsSafeFileName ensures the file name does not contain path traversal or separator chars.
func IsSafeFileName(name string) bool {
	if name == "" || strings.Contains(name, "/") || strings.Contains(name, "\\") || strings.Contains(name, "..") {
		return false
	}
	return true
}
