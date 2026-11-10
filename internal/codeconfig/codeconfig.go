// Package codeconfig reads the weave-code extension configuration.
package codeconfig

import (
	"bufio"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// Config is the extension configuration read from .weave/code.yaml.
type Config struct {
	SymbolDocuments bool
	ParseGenerated  bool
	Ignore          []string
}

// Load reads WEAVE_CODE_CONFIG or .weave/code.yaml under bundleRoot.
func Load(bundleRoot string) Config {
	config := Config{}
	if parseBool(strings.TrimSpace(os.Getenv("WEAVE_CODE_SYMBOL_DOCUMENTS"))) {
		config.SymbolDocuments = true
	}
	pathname := strings.TrimSpace(os.Getenv("WEAVE_CODE_CONFIG"))
	if pathname == "" {
		pathname = filepath.Join(bundleRoot, ".weave", "code.yaml")
	}
	file, err := os.Open(pathname)
	if err != nil {
		return config
	}
	defer file.Close()

	inIgnore := false
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := stripComment(strings.TrimSpace(scanner.Text()))
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "- ") {
			if inIgnore {
				if pattern := unquote(strings.TrimSpace(strings.TrimPrefix(line, "- "))); pattern != "" {
					config.Ignore = append(config.Ignore, pattern)
				}
			}
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key = strings.ToLower(strings.TrimSpace(key))
		value = strings.TrimSpace(value)
		inIgnore = false
		switch key {
		case "symbol_documents":
			config.SymbolDocuments = parseBool(value)
		case "parse_generated":
			config.ParseGenerated = parseBool(value)
		case "ignore":
			if value == "" {
				inIgnore = true
				continue
			}
			for _, pattern := range splitInlineList(value) {
				if pattern != "" {
					config.Ignore = append(config.Ignore, pattern)
				}
			}
		}
	}
	return config
}

// SkipReason reports why a path is excluded from analysis, or "" when it is
// eligible. Ignore rules are evaluated before source bytes are read.
func (c Config) SkipReason(relativePath string) string {
	cleaned := filepath.ToSlash(filepath.Clean(relativePath))
	if cleaned == "." || cleaned == "" {
		return ""
	}
	parts := strings.Split(cleaned, "/")
	for index, part := range parts {
		if index < len(parts)-1 && ignoredDirectory(part) {
			return "ignored directory"
		}
	}
	base := parts[len(parts)-1]
	if ignoredFile(base) {
		return "ignored file"
	}
	if IsGenerated(base) && !c.ParseGenerated {
		return "generated file"
	}
	for _, pattern := range c.Ignore {
		if matchIgnore(pattern, cleaned) {
			return "ignored path"
		}
	}
	return ""
}

// IsGenerated reports whether a file name denotes generated or minified code.
func IsGenerated(name string) bool {
	lower := strings.ToLower(name)
	return strings.Contains(lower, ".gen.") || strings.Contains(lower, ".generated.") ||
		strings.HasSuffix(lower, ".min.js") || strings.HasSuffix(lower, ".min.css")
}

func ignoredDirectory(name string) bool {
	if name == ".git" || name == "node_modules" || name == ".weave" {
		return true
	}
	if strings.HasPrefix(name, ".") {
		return true
	}
	switch name {
	case "vendor", "target", "build", "dist", "out", "coverage":
		return true
	default:
		return false
	}
}

func ignoredFile(name string) bool {
	if strings.HasPrefix(name, ".") || strings.HasSuffix(name, "~") {
		return true
	}
	switch strings.ToLower(filepath.Ext(name)) {
	case ".tmp", ".swp", ".swo":
		return true
	default:
		return false
	}
}

func matchIgnore(pattern, relativePath string) bool {
	pattern = strings.TrimPrefix(filepath.ToSlash(strings.TrimSpace(pattern)), "./")
	if pattern == "" {
		return false
	}
	if !strings.Contains(pattern, "/") {
		if matchSegments(strings.Split(pattern, "/"), strings.Split(filepath.Base(relativePath), "/")) {
			return true
		}
	}
	if matchSegments(strings.Split(pattern, "/"), strings.Split(relativePath, "/")) {
		return true
	}
	parts := strings.Split(relativePath, "/")
	for index := 1; index < len(parts); index++ {
		if matchSegments(strings.Split(pattern, "/"), parts[:index]) {
			return true
		}
	}
	return false
}

func matchSegments(pattern, target []string) bool {
	if len(pattern) == 0 {
		return len(target) == 0
	}
	if pattern[0] == "**" {
		if matchSegments(pattern[1:], target) {
			return true
		}
		if len(target) == 0 {
			return false
		}
		return matchSegments(pattern, target[1:])
	}
	if len(target) == 0 {
		return false
	}
	matched, err := path.Match(pattern[0], target[0])
	if err != nil || !matched {
		return false
	}
	return matchSegments(pattern[1:], target[1:])
}

func stripComment(line string) string {
	quoted := byte(0)
	for index := 0; index < len(line); index++ {
		switch line[index] {
		case '\'', '"':
			if quoted == 0 {
				quoted = line[index]
			} else if quoted == line[index] {
				quoted = 0
			}
		case '#':
			if quoted == 0 {
				return strings.TrimSpace(line[:index])
			}
		}
	}
	return line
}

func splitInlineList(value string) []string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "[")
	value = strings.TrimSuffix(value, "]")
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if item := unquote(strings.TrimSpace(part)); item != "" {
			result = append(result, item)
		}
	}
	return result
}

func unquote(value string) string {
	if len(value) >= 2 {
		if (value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'') {
			return value[1 : len(value)-1]
		}
	}
	return value
}

func parseBool(value string) bool {
	switch strings.ToLower(unquote(strings.TrimSpace(value))) {
	case "true", "1", "yes", "on":
		return true
	default:
		return false
	}
}
