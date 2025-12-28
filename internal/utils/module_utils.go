package utils

import (
	"bufio"
	"errors"
	"os"
	"strings"
)

// GetModuleName returns the module name from go.mod file in current directory
func GetModuleName() (string, error) {
	if _, err := os.Stat("go.mod"); os.IsNotExist(err) {
		return "", errors.New("go.mod not found")
	}

	f, err := os.Open("go.mod")
	if err != nil {
		return "", err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	// Read first few lines ensuring we find module definition
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module ")), nil
		}
	}

	return "", errors.New("module definition not found in go.mod")
}
