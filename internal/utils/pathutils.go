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

// ToGoIdentifier converts a string to a valid Go identifier (PascalCase)
func ToGoIdentifier(s string) string {
	// Split by common separators
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == '_' || r == '-' || r == ' ' || r == '.'
	})

	var result strings.Builder
	for _, part := range parts {
		if len(part) > 0 {
			// Capitalize first letter and make rest lowercase
			result.WriteString(strings.ToUpper(string(part[0])))
			if len(part) > 1 {
				result.WriteString(strings.ToLower(part[1:]))
			}
		}
	}

	identifier := result.String()
	// Ensure it starts with a letter
	if len(identifier) > 0 && !((identifier[0] >= 'A' && identifier[0] <= 'Z') || (identifier[0] >= 'a' && identifier[0] <= 'z')) {
		identifier = "Item" + identifier
	}

	return identifier
}
