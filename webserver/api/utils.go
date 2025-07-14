package api

import (
	"strings"
)

// Tries to parse the next path component from the given path.
// Returns the component and the remainder of the path.
// If no slash is found, the entire path is considered the component and remainder is empty.
func parseNextPathComponent(path string) (component string, remainder string) {
	parts := strings.SplitN(path, "/", 1)

	if len(parts) < 2 {
		return path, "" // If there's no slash, then the path is the component and there's no remainder
	}

	component = parts[0]
	remainder = parts[1]

	return component, remainder
}
