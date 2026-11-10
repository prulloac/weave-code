package main

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// symbolDocumentsEnabled reads WEAVE_CODE_SYMBOL_DOCUMENTS and .weave/code.yaml.
func symbolDocumentsEnabled(bundleRoot string) bool {
	if strings.TrimSpace(os.Getenv("WEAVE_CODE_SYMBOL_DOCUMENTS")) == "1" {
		return true
	}
	path := os.Getenv("WEAVE_CODE_CONFIG")
	if path == "" {
		path = filepath.Join(bundleRoot, ".weave", "code.yaml")
	}
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "symbol_documents:") {
			value := strings.TrimSpace(strings.TrimPrefix(line, "symbol_documents:"))
			return value == "true" || value == "1"
		}
	}
	return false
}
