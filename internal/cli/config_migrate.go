package cli

import (
	"encoding/json"
	"fmt"
	"strings"
)

func migrateConfig(data []byte, from, to string) ([]byte, error) {
	var config map[string]interface{}

	switch strings.ToLower(from) {
	case "json":
		if err := json.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("failed to parse JSON: %w", err)
		}
	case "yaml", "yml":
		config = parseYAMLFormat(string(data))
	case "toml":
		config = parseTOMLFormat(string(data))
	case "env":
		config = parseEnvFormat(string(data))
	default:
		return nil, fmt.Errorf("unsupported source format: %s", from)
	}

	switch strings.ToLower(to) {
	case "json":
		result, err := json.MarshalIndent(config, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to marshal JSON: %w", err)
		}
		return result, nil

	case "yaml", "yml":
		return []byte(generateYAMLFormat(config)), nil

	case "toml":
		return []byte(generateTOMLFormat(config)), nil

	case "env":
		return []byte(generateEnvFormat(config)), nil

	default:
		return nil, fmt.Errorf("unsupported target format: %s", to)
	}
}

func parseEnvFormat(data string) map[string]interface{} {
	config := make(map[string]interface{})
	lines := strings.Split(data, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		if (strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) ||
			(strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) {
			value = value[1 : len(value)-1]
		}

		setNestedValue(config, key, value)
	}

	return config
}

func setNestedValue(config map[string]interface{}, key, value string) {
	parts := strings.Split(strings.ToLower(key), "_")

	current := config
	for i, part := range parts {
		if i == len(parts)-1 {
			current[part] = parseValue(value)
		} else {
			if _, exists := current[part]; !exists {
				current[part] = make(map[string]interface{})
			}
			if nextMap, ok := current[part].(map[string]interface{}); ok {
				current = nextMap
			}
		}
	}
}

func parseValue(value string) interface{} {

	if strings.ToLower(value) == "true" {
		return true
	}
	if strings.ToLower(value) == "false" {
		return false
	}

	if strings.Contains(value, ".") {
		if f, err := parseFloat(value); err == nil {
			return f
		}
	} else {
		if i, err := parseInt(value); err == nil {
			return i
		}
	}

	return value
}

func parseInt(s string) (int, error) {
	result := 0
	sign := 1
	i := 0

	if i < len(s) && (s[i] == '+' || s[i] == '-') {
		if s[i] == '-' {
			sign = -1
		}
		i++
	}

	for i < len(s) {
		if s[i] < '0' || s[i] > '9' {
			return 0, fmt.Errorf("invalid character")
		}
		result = result*10 + int(s[i]-'0')
		i++
	}

	return result * sign, nil
}

func parseFloat(s string) (float64, error) {
	result := 0.0
	sign := 1.0
	i := 0

	if i < len(s) && (s[i] == '+' || s[i] == '-') {
		if s[i] == '-' {
			sign = -1
		}
		i++
	}

	for i < len(s) && s[i] != '.' {
		if s[i] < '0' || s[i] > '9' {
			return 0, fmt.Errorf("invalid character")
		}
		result = result*10 + float64(s[i]-'0')
		i++
	}

	if i < len(s) && s[i] == '.' {
		i++
		decimal := 0.1
		for i < len(s) {
			if s[i] < '0' || s[i] > '9' {
				return 0, fmt.Errorf("invalid character")
			}
			result += float64(s[i]-'0') * decimal
			decimal *= 0.1
			i++
		}
	}

	return result * sign, nil
}

func generateEnvFormat(config map[string]interface{}) string {
	var lines []string
	flattenConfig(config, "", &lines)
	return strings.Join(lines, "\n")
}

func flattenConfig(config map[string]interface{}, prefix string, lines *[]string) {
	for key, value := range config {
		envKey := key
		if prefix != "" {
			envKey = prefix + "_" + key
		}
		envKey = strings.ToUpper(envKey)

		switch v := value.(type) {
		case map[string]interface{}:
			flattenConfig(v, envKey, lines)
		case []interface{}:
			var strValues []string
			for _, item := range v {
				strValues = append(strValues, fmt.Sprintf("%v", item))
			}
			*lines = append(*lines, fmt.Sprintf("%s=%s", envKey, strings.Join(strValues, ",")))
		default:
			*lines = append(*lines, fmt.Sprintf("%s=%v", envKey, formatEnvValue(v)))
		}
	}
}

func formatEnvValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		if strings.ContainsAny(v, " \t\n\r\"'\\") {
			return fmt.Sprintf(`"%s"`, strings.ReplaceAll(v, `"`, `\"`))
		}
		return v
	case bool:
		if v {
			return "true"
		}
		return "false"
	default:
		return fmt.Sprintf("%v", v)
	}
}

// simple YAML parser (basic implementatation, I dont want to use external lib for that)
func parseYAMLFormat(data string) map[string]interface{} {
	config := make(map[string]interface{})
	lines := strings.Split(data, "\n")

	var currentMap map[string]interface{} = config

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		indent := 0
		for _, char := range line {
			if char == ' ' {
				indent++
			} else {
				break
			}
		}

		if strings.Contains(line, ":") {
			parts := strings.SplitN(strings.TrimSpace(line), ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])

				if value == "" {
					nested := make(map[string]interface{})
					currentMap[key] = nested
				} else {
					currentMap[key] = parseValue(value)
				}
			}
		}
	}

	return config
}

// Simple TOML parser (basic implementation, idem than above)
func parseTOMLFormat(data string) map[string]interface{} {
	config := make(map[string]interface{})
	lines := strings.Split(data, "\n")

	currentSection := config

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section := line[1 : len(line)-1]
			sectionMap := make(map[string]interface{})
			config[section] = sectionMap
			currentSection = sectionMap
			continue
		}

		if strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])

				if (strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) ||
					(strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) {
					value = value[1 : len(value)-1]
				}

				currentSection[key] = parseValue(value)
			}
		}
	}

	return config
}

func generateYAMLFormat(config map[string]interface{}) string {
	var lines []string
	generateYAMLLines(config, "", &lines)
	return strings.Join(lines, "\n")
}

func generateYAMLLines(config map[string]interface{}, prefix string, lines *[]string) {
	for key, value := range config {
		switch v := value.(type) {
		case map[string]interface{}:
			*lines = append(*lines, prefix+key+":")
			generateYAMLLines(v, prefix+"  ", lines)
		case []interface{}:
			*lines = append(*lines, prefix+key+":")
			for _, item := range v {
				*lines = append(*lines, prefix+"  - "+fmt.Sprintf("%v", item))
			}
		default:
			*lines = append(*lines, fmt.Sprintf("%s%s: %v", prefix, key, formatYAMLValue(v)))
		}
	}
}

func formatYAMLValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		if strings.ContainsAny(v, " \t\n\r\"'\\:") {
			return fmt.Sprintf(`"%s"`, strings.ReplaceAll(v, `"`, `\"`))
		}
		return v
	case bool:
		if v {
			return "true"
		}
		return "false"
	default:
		return fmt.Sprintf("%v", v)
	}
}

func generateTOMLFormat(config map[string]interface{}) string {
	var lines []string

	for key, value := range config {
		if _, isMap := value.(map[string]interface{}); !isMap {
			lines = append(lines, fmt.Sprintf("%s = %s", key, formatTOMLValue(value)))
		}
	}

	for key, value := range config {
		if subMap, isMap := value.(map[string]interface{}); isMap {
			if len(lines) > 0 {
				lines = append(lines, "") // Add empty line before section
			}
			lines = append(lines, fmt.Sprintf("[%s]", key))
			for subKey, subValue := range subMap {
				lines = append(lines, fmt.Sprintf("%s = %s", subKey, formatTOMLValue(subValue)))
			}
		}
	}

	return strings.Join(lines, "\n")
}

func formatTOMLValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		return fmt.Sprintf(`"%s"`, strings.ReplaceAll(v, `"`, `\"`))
	case bool:
		if v {
			return "true"
		}
		return "false"
	case []interface{}:
		var items []string
		for _, item := range v {
			items = append(items, fmt.Sprintf(`"%v"`, item))
		}
		return fmt.Sprintf("[%s]", strings.Join(items, ", "))
	default:
		return fmt.Sprintf("%v", v)
	}
}
