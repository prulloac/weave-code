package lexical

import (
	"regexp"
	"strings"

	"github.com/prulloac/weave-code/internal/model"
)

var (
	functionPattern = regexp.MustCompile(`(?m)^\s*(?:func|def|fn|function|class|struct|interface|type)\s+([A-Za-z_][A-Za-z0-9_]*)`)
	importPattern   = regexp.MustCompile(`(?m)^\s*(?:import|use|require)\s+["']?([A-Za-z0-9_./:-]+)`)
)

// Parse extracts best-effort metadata from an unparseable file.
func Parse(file model.File, source []byte, language string) model.Unit {
	text := string(source)
	unit := model.Unit{
		File:     file,
		Language: language,
		Parser:   "lexical",
		Status:   "fallback",
		Warnings: []string{"parser failed; lexical fallback used"},
	}
	for _, match := range functionPattern.FindAllStringSubmatch(text, -1) {
		name := match[1]
		unit.Symbols = append(unit.Symbols, model.Symbol{
			Name:       name,
			Qualified:  name,
			Kind:       "symbol",
			Signature:  name,
			Visibility: visibility(name),
		})
	}
	for _, match := range importPattern.FindAllStringSubmatch(text, -1) {
		if imported := strings.TrimSpace(match[1]); imported != "" {
			unit.Imports = append(unit.Imports, model.Import{Path: imported})
		}
	}
	return unit
}

func visibility(name string) string {
	if name == "" || strings.HasPrefix(name, "_") || strings.HasPrefix(name, "lower") {
		return "private"
	}
	return "public"
}
