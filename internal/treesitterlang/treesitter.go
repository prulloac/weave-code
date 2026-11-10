package treesitterlang

import (
	"regexp"
	"strings"

	"github.com/odvcencio/gotreesitter"
	cgrammar "github.com/odvcencio/gotreesitter/grammars/c"
	cppgrammar "github.com/odvcencio/gotreesitter/grammars/cpp"
	dartgrammar "github.com/odvcencio/gotreesitter/grammars/dart"
	javagrammar "github.com/odvcencio/gotreesitter/grammars/java"
	jsgrammar "github.com/odvcencio/gotreesitter/grammars/javascript"
	rustgrammar "github.com/odvcencio/gotreesitter/grammars/rust"
	scalagrammar "github.com/odvcencio/gotreesitter/grammars/scala"
	tsgrammar "github.com/odvcencio/gotreesitter/grammars/typescript"
	ziggrammar "github.com/odvcencio/gotreesitter/grammars/zig"

	"github.com/prulloac/weave-code/internal/model"
)

var (
	javaPackagePattern  = regexp.MustCompile(`(?m)^\s*package\s+([A-Za-z0-9_.]+)`)
	scalaPackagePattern = regexp.MustCompile(`(?m)^\s*package\s+([A-Za-z0-9_.]+)`)
	cIncludePattern     = regexp.MustCompile(`(?m)^\s*#\s*include\s+[<"]([^>"]+)[>"]`)
	jsImportPattern     = regexp.MustCompile(`(?m)^\s*import\s+(?:[^'"]+\s+from\s+)?["']([^"']+)["']`)
	jsRequirePattern    = regexp.MustCompile(`require\(\s*["']([^"']+)["']\s*\)`)
	rustUsePattern      = regexp.MustCompile(`(?m)^\s*use\s+([A-Za-z0-9_:{}, *]+);`)
	javaImportPattern   = regexp.MustCompile(`(?m)^\s*import\s+(?:static\s+)?([A-Za-z0-9_.*]+);`)
	scalaImportPattern  = regexp.MustCompile(`(?m)^\s*import\s+([A-Za-z0-9_.{}, *]+)`)
	dartImportPattern   = regexp.MustCompile(`(?m)^\s*import\s+["']([^"']+)["']`)
	zigImportPattern    = regexp.MustCompile(`@import\(\s*["']([^"']+)["']\s*\)`)
	identifierPattern   = regexp.MustCompile(`[A-Za-z_][A-Za-z0-9_]*`)
	baseKindParts       = []string{"extends", "implements", "superclass", "super_interfaces", "base_class", "base_list", "inheritance", "heritage", "impl_item", "interfaces"}
	baseNameBlacklist   = map[string]bool{"extends": true, "implements": true, "class": true, "interface": true, "superclass": true, "with": true, "for": true, "public": true, "private": true, "protected": true, "virtual": true, "abstract": true}
	reservedNames       = map[string]bool{"func": true, "fn": true, "def": true, "function": true, "class": true, "struct": true, "union": true, "interface": true, "trait": true, "enum": true, "record": true, "object": true, "mixin": true, "type": true, "const": true, "static": true, "void": true}
)

var symbolKinds = map[string]string{
	"function_declaration":           "function",
	"generator_function_declaration": "function",
	"function_definition":            "function",
	"function_item":                  "function",
	"function_signature":             "function",
	"method_declaration":             "function",
	"method_definition":              "function",
	"constructor_declaration":        "function",
	"fn_declaration":                 "function",
	"class_declaration":              "type",
	"class_definition":               "type",
	"class_specifier":                "type",
	"struct_specifier":               "type",
	"struct_item":                    "type",
	"struct_declaration":             "type",
	"union_declaration":              "type",
	"interface_declaration":          "type",
	"trait_item":                     "type",
	"trait_definition":               "type",
	"enum_declaration":               "type",
	"enum_item":                      "type",
	"enum_specifier":                 "type",
	"record_declaration":             "type",
	"object_definition":              "type",
	"mixin_declaration":              "type",
	"type_item":                      "type",
	"type_declaration":               "type",
	"type_alias_declaration":         "type",
}

var callKinds = map[string]bool{
	"call_expression":                true,
	"method_invocation":              true,
	"object_creation_expression":     true,
	"function_expression_invocation": true,
	"call":                           true,
}

var referenceKinds = map[string]bool{
	"identifier":          true,
	"type_identifier":     true,
	"property_identifier": true,
	"field_identifier":    true,
}

type languageSpec struct {
	name         string
	language     *gotreesitter.Language
	packageRegex *regexp.Regexp
	importRegex  []*regexp.Regexp
}

func specFor(language string) (languageSpec, bool) {
	switch language {
	case "javascript":
		return languageSpec{name: "javascript", language: jsgrammar.Language(), importRegex: []*regexp.Regexp{jsImportPattern, jsRequirePattern}}, true
	case "typescript":
		return languageSpec{name: "typescript", language: tsgrammar.Language(), importRegex: []*regexp.Regexp{jsImportPattern, jsRequirePattern}}, true
	case "rust":
		return languageSpec{name: "rust", language: rustgrammar.Language(), importRegex: []*regexp.Regexp{rustUsePattern}}, true
	case "c":
		return languageSpec{name: "c", language: cgrammar.Language(), importRegex: []*regexp.Regexp{cIncludePattern}}, true
	case "c++":
		return languageSpec{name: "c++", language: cppgrammar.Language(), importRegex: []*regexp.Regexp{cIncludePattern}}, true
	case "java":
		return languageSpec{name: "java", language: javagrammar.Language(), packageRegex: javaPackagePattern, importRegex: []*regexp.Regexp{javaImportPattern}}, true
	case "scala":
		return languageSpec{name: "scala", language: scalagrammar.Language(), packageRegex: scalaPackagePattern, importRegex: []*regexp.Regexp{scalaImportPattern}}, true
	case "dart":
		return languageSpec{name: "dart", language: dartgrammar.Language(), importRegex: []*regexp.Regexp{dartImportPattern}}, true
	case "zig":
		return languageSpec{name: "zig", language: ziggrammar.Language(), importRegex: []*regexp.Regexp{zigImportPattern}}, true
	}
	return languageSpec{}, false
}

// Parse parses one source file with a pure-Go tree-sitter grammar.
func Parse(file model.File, source []byte, language string) (model.Unit, bool) {
	spec, ok := specFor(language)
	if !ok {
		return model.Unit{}, false
	}
	parser := gotreesitter.NewParser(spec.language)
	tree, err := parser.Parse(source)
	if err != nil {
		return model.Unit{}, false
	}
	root := tree.RootNode()
	unit := model.Unit{
		File:     file,
		Language: spec.name,
		Module:   moduleName(file.Path),
		Parser:   "tree-sitter-" + spec.name,
		Status:   "parsed",
	}
	if spec.packageRegex != nil {
		if match := spec.packageRegex.FindStringSubmatch(string(source)); len(match) == 2 {
			unit.Package = match[1]
			unit.Module = match[1]
		}
	}
	if root.HasError() {
		unit.Status = "partial"
		unit.Warnings = append(unit.Warnings, "parse tree contains syntax errors")
	}
	walk(root, source, spec, &unit, "", "")
	unit.Imports = parseImports(source, spec.importRegex)
	return unit, true
}

func walk(node *gotreesitter.Node, source []byte, spec languageSpec, unit *model.Unit, parent, current string) string {
	if node == nil {
		return current
	}
	kind := node.Type(spec.language)
	if symbolKind, ok := symbolKinds[kind]; ok {
		name := symbolName(node, source, spec.language)
		if name != "" {
			symbolName := name
			if parent != "" {
				symbolName = parent + "." + name
			}
			qualified := symbolName
			if unit.Package != "" {
				qualified = unit.Package + "." + symbolName
			} else if unit.Module != "" {
				qualified = unit.Module + "." + symbolName
			}
			signature := firstLine(node.Text(source))
			unit.Symbols = append(unit.Symbols, model.Symbol{
				Name:       name,
				Qualified:  qualified,
				Kind:       symbolKind,
				Signature:  signature,
				Parent:     parent,
				Visibility: visibility(name),
				LineStart:  int(node.StartPoint().Row + 1),
				LineEnd:    int(node.EndPoint().Row + 1),
			})
			parent = symbolName
			current = qualified
		}
	}
	if callKinds[node.Type(spec.language)] {
		if current != "" {
			if callee := callName(node, source, spec.language); callee != "" {
				unit.Calls = append(unit.Calls, model.Reference{From: current, Name: callee})
			}
		}
	}
	if spec.name == "dart" && current != "" {
		for _, callee := range dartInvocations(node, source, spec.language) {
			unit.Calls = append(unit.Calls, model.Reference{From: current, Name: callee})
		}
	}
	if referenceKinds[kind] && current != "" {
		name := strings.TrimSpace(node.Text(source))
		if name != "" && !reservedNames[name] {
			unit.References = append(unit.References, model.Reference{From: current, Name: name, Kind: "references"})
		}
	}
	if isBaseKind(node.Type(spec.language)) && current != "" {
		relationKind := baseRelationKind(node.Type(spec.language))
		names := baseNames(node.Text(source))
		if node.Type(spec.language) == "impl_item" {
			names = rustImplTraits(node.Text(source))
		}
		for _, name := range names {
			unit.Bases = append(unit.Bases, model.Reference{From: current, Name: name, Kind: relationKind})
		}
	}
	for index := 0; index < node.NamedChildCount(); index++ {
		current = walk(node.NamedChild(index), source, spec, unit, childParent(kind, parent, node.Text(source)), current)
	}
	return current
}

func childParent(kind, parent, text string) string {
	if kind == "impl_item" {
		if receiver := rustImplReceiver(text); receiver != "" {
			return receiver
		}
	}
	return parent
}

func isBaseKind(kind string) bool {
	for _, part := range baseKindParts {
		if strings.Contains(kind, part) {
			return true
		}
	}
	return false
}

func baseRelationKind(kind string) string {
	if strings.Contains(kind, "implements") || strings.Contains(kind, "interfaces") || kind == "impl_item" {
		return "implements"
	}
	return "inherits"
}

func rustImplTraits(text string) []string {
	text = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(text), "impl"))
	index := strings.Index(text, " for ")
	if index < 0 {
		return nil
	}
	if name := lastIdentifier(text[:index]); name != "" {
		return []string{name}
	}
	return nil
}

func rustImplReceiver(text string) string {
	text = strings.TrimSpace(text)
	text = strings.TrimPrefix(text, "impl")
	if index := strings.Index(text, " for "); index >= 0 {
		text = text[index+len(" for "):]
	}
	if index := strings.IndexByte(text, '{'); index >= 0 {
		text = text[:index]
	}
	text = strings.TrimSpace(text)
	if strings.HasPrefix(text, "<") {
		if index := strings.IndexByte(text, '>'); index >= 0 {
			text = strings.TrimSpace(text[index+1:])
		}
	}
	if index := strings.Index(text, "where"); index >= 0 {
		text = strings.TrimSpace(text[:index])
	}
	if index := strings.IndexByte(text, '<'); index >= 0 {
		text = strings.TrimSpace(text[:index])
	}
	return lastIdentifier(text)
}

func symbolName(node *gotreesitter.Node, source []byte, language *gotreesitter.Language) string {
	for _, field := range []string{"name", "declarator", "type"} {
		if child := node.ChildByFieldName(field, language); child != nil {
			if name := firstIdentifier(child.Text(source)); name != "" && !reservedNames[name] {
				return name
			}
		}
	}
	name := firstIdentifier(node.Text(source))
	if reservedNames[name] {
		return ""
	}
	return name
}

func callName(node *gotreesitter.Node, source []byte, language *gotreesitter.Language) string {
	for _, field := range []string{"function", "method", "name"} {
		if child := node.ChildByFieldName(field, language); child != nil {
			if name := lastIdentifier(child.Text(source)); name != "" {
				return name
			}
		}
	}
	if node.NamedChildCount() > 0 {
		return lastIdentifier(node.NamedChild(0).Text(source))
	}
	return ""
}

func dartInvocations(node *gotreesitter.Node, source []byte, language *gotreesitter.Language) []string {
	children := node.NamedChildCount()
	if children < 2 {
		return nil
	}
	result := make([]string, 0)
	for index := 1; index < children; index++ {
		child := node.NamedChild(index)
		if child.Type(language) != "selector" {
			continue
		}
		text := child.Text(source)
		if !strings.Contains(text, "(") {
			continue
		}
		name := firstIdentifier(text)
		if name == "" {
			previous := node.NamedChild(index - 1)
			switch previous.Type(language) {
			case "identifier", "type_identifier":
				name = strings.TrimSpace(previous.Text(source))
			}
		}
		if name != "" {
			result = append(result, name)
		}
	}
	return result
}

func baseNames(text string) []string {
	result := make([]string, 0)
	for _, match := range identifierPattern.FindAllString(text, -1) {
		if baseNameBlacklist[match] {
			continue
		}
		result = append(result, match)
	}
	return result
}

func parseImports(source []byte, patterns []*regexp.Regexp) []model.Import {
	seen := map[string]bool{}
	result := make([]model.Import, 0)
	for _, pattern := range patterns {
		for _, match := range pattern.FindAllStringSubmatch(string(source), -1) {
			if len(match) < 2 {
				continue
			}
			path := strings.TrimSpace(match[1])
			if path == "" || seen[path] {
				continue
			}
			seen[path] = true
			result = append(result, model.Import{Path: path})
		}
	}
	return result
}

func firstIdentifier(text string) string {
	match := identifierPattern.FindString(text)
	return match
}

func lastIdentifier(text string) string {
	matches := identifierPattern.FindAllString(text, -1)
	if len(matches) == 0 {
		return ""
	}
	return matches[len(matches)-1]
}

func firstLine(text string) string {
	text = strings.TrimSpace(text)
	if index := strings.IndexByte(text, '\n'); index >= 0 {
		text = text[:index]
	}
	if len(text) > 200 {
		text = text[:200]
	}
	return strings.TrimSpace(text)
}

func moduleName(path string) string {
	path = strings.TrimSuffix(strings.ReplaceAll(path, "\\", "/"), extension(path))
	return strings.ReplaceAll(path, "/", ".")
}

func extension(path string) string {
	if index := strings.LastIndexByte(path, '.'); index >= 0 {
		return path[index:]
	}
	return ""
}

func visibility(name string) string {
	if name == "" || strings.HasPrefix(name, "_") {
		return "private"
	}
	return "public"
}
