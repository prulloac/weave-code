package analyze

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/prulloac/weave-code/internal/codeconfig"
	"github.com/prulloac/weave-code/internal/golanglang"
	"github.com/prulloac/weave-code/internal/lexical"
	"github.com/prulloac/weave-code/internal/model"
	"github.com/prulloac/weave-code/internal/pythonlang"
	"github.com/prulloac/weave-code/internal/treesitterlang"
)

const (
	maxSourceFiles      = 10_000
	maxProjectBytes     = 512 << 20
	maxSourceFileBytes  = 4 << 20
	maxSymbolsPerFile   = 2_000
	maxProjectRelations = 1_000_000
	maxDocCommentBytes  = 64 << 10
)

type symbolRef struct {
	fileID   string
	name     string
	language string
}

// Run analyzes every requested file and returns the project response.
func Run(request model.Request) (model.Response, error) {
	files := request.Files
	response := model.Response{V: "weave.cli/v1", Documents: make([]model.Document, 0, len(files))}
	if len(files) > maxSourceFiles {
		response.Warnings = append(response.Warnings, fmt.Sprintf("project file list truncated to %d files", maxSourceFiles))
		files = files[:maxSourceFiles]
	}
	config := codeconfig.Config{
		ParseGenerated: request.Options.ParseGenerated,
		Ignore:         request.Options.Ignore,
	}
	units := make([]model.Unit, 0, len(files))
	var totalBytes int64
	for _, file := range files {
		if reason := config.SkipReason(file.Path); reason != "" {
			response.Warnings = append(response.Warnings, fmt.Sprintf("%s: skipped (%s)", file.ID, reason))
			continue
		}
		path := filepath.Join(request.BundleRoot, filepath.FromSlash(file.Path))
		info, err := os.Stat(path)
		if err != nil {
			response.Warnings = append(response.Warnings, fmt.Sprintf("%s: %v", file.ID, err))
			units = append(units, model.Unit{File: file, Status: "fallback", Parser: "none"})
			continue
		}
		if info.Size() > maxSourceFileBytes {
			response.Warnings = append(response.Warnings, fmt.Sprintf("%s: skipped (file exceeds 4 MiB)", file.ID))
			continue
		}
		if totalBytes+info.Size() > maxProjectBytes {
			response.Warnings = append(response.Warnings, fmt.Sprintf("%s: skipped (project exceeds 512 MiB)", file.ID))
			continue
		}
		source, err := os.ReadFile(path)
		if err != nil {
			response.Warnings = append(response.Warnings, fmt.Sprintf("%s: %v", file.ID, err))
			units = append(units, model.Unit{File: file, Status: "fallback", Parser: "none"})
			continue
		}
		if !utf8.Valid(source) || bytes.IndexByte(source, 0) >= 0 {
			response.Warnings = append(response.Warnings, fmt.Sprintf("%s: skipped (invalid UTF-8 or NUL byte)", file.ID))
			continue
		}
		totalBytes += int64(len(source))
		unit := parseFile(file, source)
		if len(unit.Symbols) > maxSymbolsPerFile {
			response.Warnings = append(response.Warnings, fmt.Sprintf("%s: symbol documents truncated to %d", file.ID, maxSymbolsPerFile))
			unit.Symbols = unit.Symbols[:maxSymbolsPerFile]
		}
		units = append(units, unit)
	}

	qualified := make(map[string]symbolRef)
	bare := make(map[string][]symbolRef)
	for unitIndex := range units {
		unit := &units[unitIndex]
		for _, symbol := range unit.Symbols {
			ref := symbolRef{fileID: unit.File.ID, name: symbolPath(*unit, symbol), language: unit.Language}
			qualified[strings.ToLower(symbol.Qualified)] = ref
			key := strings.ToLower(symbol.Name)
			bare[key] = append(bare[key], ref)
		}
	}

	external := make(map[string][]model.Relation)
	externalMeta := make(map[string]model.Import)
	for unitIndex := range units {
		unit := &units[unitIndex]
		document := fileDocument(*unit)
		if request.Options.SymbolDocuments {
			for _, symbol := range unit.Symbols {
				document.SymbolDocuments = append(document.SymbolDocuments, symbolDocument(*unit, symbol))
				response.Relations = append(response.Relations, model.Relation{
					From: unit.File.ID, To: symbolID(*unit, symbol), Kind: "declares",
				})
			}
		}
		response.Documents = append(response.Documents, document)
		for _, imported := range unit.Imports {
			if target, ok := resolveImport(imported.Path, units); ok {
				response.Relations = append(response.Relations, model.Relation{
					From: unit.File.ID, To: target, Kind: "imports",
				})
				continue
			}
			id := externalID(unit.Language, imported.Path)
			if id == "" {
				continue
			}
			external[id] = append(external[id], model.Relation{From: unit.File.ID, To: id, Kind: "imports"})
			externalMeta[id] = imported
		}
		for _, call := range unit.Calls {
			from := relationSource(*unit, call.From, request.Options.SymbolDocuments)
			target, ok := resolveReference(call.Name, qualified, bare, *unit)
			if !ok {
				continue
			}
			to := relationTarget(*unit, target, units, request.Options.SymbolDocuments)
			if to == "" || from == to {
				continue
			}
			response.Relations = append(response.Relations, model.Relation{From: from, To: to, Kind: "calls"})
		}
		for _, reference := range unit.References {
			from := relationSource(*unit, reference.From, request.Options.SymbolDocuments)
			target, ok := resolveReference(reference.Name, qualified, bare, *unit)
			if !ok {
				continue
			}
			to := relationTarget(*unit, target, units, request.Options.SymbolDocuments)
			if to == "" || from == to {
				continue
			}
			kind := reference.Kind
			if kind == "" {
				kind = "references"
			}
			response.Relations = append(response.Relations, model.Relation{From: from, To: to, Kind: kind})
		}
		for _, base := range unit.Bases {
			from := relationSource(*unit, base.From, request.Options.SymbolDocuments)
			target, ok := resolveReference(base.Name, qualified, bare, *unit)
			if !ok {
				continue
			}
			to := relationTarget(*unit, target, units, request.Options.SymbolDocuments)
			if to == "" || from == to {
				continue
			}
			kind := base.Kind
			if kind == "" {
				kind = "inherits"
			}
			response.Relations = append(response.Relations, model.Relation{From: from, To: to, Kind: kind})
		}
	}
	for id, relations := range external {
		if len(relations) <= 1 {
			continue
		}
		imported := externalMeta[id]
		language := strings.Split(strings.TrimPrefix(id, "_code/external/"), "/")[0]
		title := imported.Path
		response.ExternalDocuments = append(response.ExternalDocuments, model.Document{
			ID:       id,
			Title:    title,
			Keywords: []string{language, imported.Path},
			Body:     imported.Path,
			ContentMetadata: map[string]any{
				"v": 1, "language": language, "package": imported.Path, "references": len(relations),
			},
		})
		response.Relations = append(response.Relations, relations...)
	}
	response.Relations = dedupe(response.Relations)
	sort.Slice(response.Relations, func(i, j int) bool {
		if response.Relations[i].From != response.Relations[j].From {
			return response.Relations[i].From < response.Relations[j].From
		}
		if response.Relations[i].To != response.Relations[j].To {
			return response.Relations[i].To < response.Relations[j].To
		}
		return response.Relations[i].Kind < response.Relations[j].Kind
	})
	if len(response.Relations) > maxProjectRelations {
		response.Warnings = append(response.Warnings, fmt.Sprintf("relations truncated to %d", maxProjectRelations))
		response.Relations = response.Relations[:maxProjectRelations]
	}
	return response, nil
}

func parseFile(file model.File, source []byte) model.Unit {
	language := languageFor(file.Path)
	switch language {
	case "go":
		unit, err := golanglang.Parse(file, source)
		if err == nil {
			return unit
		}
	case "python":
		unit, err := pythonlang.Parse(file, source)
		if err == nil {
			return unit
		}
	default:
		if unit, ok := treesitterlang.Parse(file, source, language); ok {
			return unit
		}
	}
	return lexical.Parse(file, source, language)
}

func languageFor(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".go":
		return "go"
	case ".py":
		return "python"
	case ".js", ".jsx":
		return "javascript"
	case ".ts", ".tsx":
		return "typescript"
	case ".rs":
		return "rust"
	case ".c", ".h":
		return "c"
	case ".cc", ".cpp", ".cxx", ".hpp", ".hh", ".hxx":
		return "c++"
	case ".java":
		return "java"
	case ".scala", ".sc":
		return "scala"
	case ".dart":
		return "dart"
	case ".zig":
		return "zig"
	default:
		return ""
	}
}

func fileDocument(unit model.Unit) model.Document {
	title := unit.Module
	if unit.Package != "" {
		title = unit.Package + " package"
	} else if title == "" {
		title = unit.File.ID
	}
	keywords := keywords(unit)
	return model.Document{
		ID:          unit.File.ID,
		Title:       title,
		Description: truncate(unit.Doc, 1024),
		Keywords:    keywords,
		ContentMetadata: map[string]any{
			"v":        1,
			"language": unit.Language,
			"package":  unit.Package,
			"module":   unit.Module,
			"parser":   unit.Parser,
			"status":   unit.Status,
			"symbols":  len(unit.Symbols),
			"imports":  len(unit.Imports),
		},
	}
}

func symbolDocument(unit model.Unit, symbol model.Symbol) model.Document {
	body := symbol.Signature
	if symbol.Doc != "" {
		body += "\n\n" + truncate(symbol.Doc, maxDocCommentBytes)
	}
	return model.Document{
		ID:          symbolID(unit, symbol),
		Title:       symbolPath(unit, symbol),
		Description: truncate(symbol.Doc, 1024),
		Keywords:    []string{unit.Language, symbol.Kind, symbol.Name},
		Body:        body,
		ContentMetadata: map[string]any{
			"v": 1, "language": unit.Language, "kind": symbol.Kind, "signature": symbol.Signature,
			"visibility": symbol.Visibility, "parent": symbol.Parent,
			"line_start": symbol.LineStart, "line_end": symbol.LineEnd,
		},
	}
}

func keywords(unit model.Unit) []string {
	seen := map[string]bool{}
	result := make([]string, 0)
	add := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] || len(result) >= 64 {
			return
		}
		seen[value] = true
		result = append(result, value)
	}
	add(unit.Language)
	add(unit.Package)
	add(unit.Module)
	for _, symbol := range unit.Symbols {
		add(symbol.Name)
	}
	for _, imported := range unit.Imports {
		add(imported.Path)
	}
	return result
}

func symbolPath(unit model.Unit, symbol model.Symbol) string {
	prefix := unit.Package
	if unit.Module != "" {
		prefix = unit.Module
	}
	if prefix != "" && strings.HasPrefix(symbol.Qualified, prefix+".") {
		return strings.TrimPrefix(symbol.Qualified, prefix+".")
	}
	return symbol.Name
}

func symbolID(unit model.Unit, symbol model.Symbol) string {
	return unit.File.ID + "#" + symbolPath(unit, symbol)
}

func relationSource(unit model.Unit, qualifiedName string, symbols bool) string {
	if !symbols {
		return unit.File.ID
	}
	for _, symbol := range unit.Symbols {
		if symbol.Qualified == qualifiedName {
			return symbolID(unit, symbol)
		}
	}
	return unit.File.ID
}

func relationTarget(source model.Unit, target symbolRef, units []model.Unit, symbols bool) string {
	if symbols {
		unit := unitByID(units, target.fileID)
		for _, symbol := range unit.Symbols {
			if symbolPath(unit, symbol) == target.name {
				return symbolID(unit, symbol)
			}
		}
	}
	return target.fileID
}

func unitByID(units []model.Unit, id string) model.Unit {
	for _, unit := range units {
		if unit.File.ID == id {
			return unit
		}
	}
	return model.Unit{File: model.File{ID: id}}
}

func resolveReference(name string, qualified map[string]symbolRef, bare map[string][]symbolRef, source model.Unit) (symbolRef, bool) {
	key := strings.ToLower(strings.TrimSpace(name))
	if key == "" {
		return symbolRef{}, false
	}
	if ref, ok := qualified[key]; ok {
		return ref, true
	}
	matches := bare[key]
	if len(matches) == 1 {
		return matches[0], true
	}
	if len(matches) == 0 {
		return symbolRef{}, false
	}
	if ref, ok := uniqueMatch(matches, func(ref symbolRef) bool {
		return ref.fileID == source.File.ID
	}); ok {
		return ref, true
	}
	if ref, ok := uniqueMatch(matches, func(ref symbolRef) bool {
		return ref.language == source.Language
	}); ok {
		return ref, true
	}
	return symbolRef{}, false
}

func uniqueMatch(matches []symbolRef, keep func(symbolRef) bool) (symbolRef, bool) {
	var found symbolRef
	count := 0
	for _, ref := range matches {
		if keep(ref) {
			found = ref
			count++
		}
	}
	return found, count == 1
}

func resolveImport(path string, units []model.Unit) (string, bool) {
	cleaned := strings.Trim(path, `"'`)
	if cleaned == "" {
		return "", false
	}
	trimmed := strings.TrimPrefix(cleaned, ".")
	last := trimmed
	if index := strings.LastIndex(last, "/"); index >= 0 {
		last = last[index+1:]
	}
	if index := strings.LastIndex(last, "."); index >= 0 && !strings.Contains(last, "/") {
		last = last[index+1:]
	}
	for _, unit := range units {
		if unit.Package == last || unit.Module == trimmed || strings.HasSuffix(unit.Module, "."+trimmed) {
			return unit.File.ID, true
		}
	}
	return "", false
}

func externalID(language, path string) string {
	path = strings.Trim(strings.TrimSpace(path), `"'`)
	path = strings.TrimPrefix(path, ".")
	if path == "" {
		return ""
	}
	return "_code/external/" + language + "/" + strings.ReplaceAll(path, " ", "_")
}

func dedupe(relations []model.Relation) []model.Relation {
	seen := map[string]bool{}
	result := make([]model.Relation, 0, len(relations))
	for _, relation := range relations {
		key := relation.From + "\x00" + relation.To + "\x00" + relation.Kind
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, relation)
	}
	return result
}

func truncate(text string, limit int) string {
	if len(text) <= limit {
		return text
	}
	return text[:limit]
}
