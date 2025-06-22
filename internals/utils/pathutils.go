package utils

import (
	"strings"
)

// MatchPath matches URL paths with patterns like "/users/:id"
// Need a rewrite later on.

func MatchPath(pattern, path string) bool {
	patternParts := strings.Split(pattern, "/")
	pathParts := strings.Split(path, "/")

	if len(patternParts) != len(pathParts) {
		// Need to handler special case for wildcard paths like "/static/*filepath"
		// MEMO: other special case to handle ??
		if len(patternParts) > 0 && strings.HasPrefix(patternParts[len(patternParts)-1], "*") {
			return strings.HasPrefix(path, strings.Join(patternParts[:len(patternParts)-1], "/"))
		}
		return false
	}

	for i := 0; i < len(patternParts); i++ {
		if patternParts[i] == "" && pathParts[i] == "" {
			continue
		}

		if patternParts[i] != pathParts[i] && !strings.HasPrefix(patternParts[i], ":") && !strings.HasPrefix(patternParts[i], "*") {
			return false
		}
	}
	return true
}

// extract the params from the path
func ExtractParams(pattern, path string) map[string]string {
	params := make(map[string]string)
	patternParts := strings.Split(pattern, "/")
	pathParts := strings.Split(path, "/")
	// MEMO: that's ugly af wanna rewwrite
	for i := 0; i < len(patternParts) && i < len(pathParts); i++ {
		if strings.HasPrefix(patternParts[i], ":") {
			paramName := patternParts[i][1:] // remove the : prefix
			params[paramName] = pathParts[i]
		} else if strings.HasPrefix(patternParts[i], "*") {
			paramName := patternParts[i][1:] // remove the * prefix
			params[paramName] = strings.Join(pathParts[i:], "/")
			break
		}
	}
	return params
}
