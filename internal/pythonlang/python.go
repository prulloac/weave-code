package pythonlang

import (
	"strings"

	bonsaipython "github.com/msuozzo/bonsai/bonsai-python"

	"github.com/prulloac/weave-code/internal/model"
)

// Parse parses one Python source file.
func Parse(file model.File, source []byte) (model.Unit, error) {
	parser := bonsaipython.NewParser()
	root, err := parser.Parse(source)
	if err != nil {
		return model.Unit{}, err
	}
	unit := model.Unit{
		File:     file,
		Language: "python",
		Module:   moduleName(file.Path),
		Parser:   "tree-sitter-python",
		Status:   "parsed",
	}
	if root.HasError() {
		unit.Status = "partial"
		unit.Warnings = append(unit.Warnings, "parse tree contains syntax errors")
	}
	walk(root, source, &unit, "", "")
	return unit, nil
}

func walk(node *bonsaipython.Node, source []byte, unit *model.Unit, parent string, currentFunction string) {
	switch node.Kind {
	case bonsaipython.KindDecoratedDefinition:
		for _, child := range node.Children {
			if child.Kind == bonsaipython.KindFunctionDefinition || child.Kind == bonsaipython.KindClassDefinition {
				walk(child, source, unit, parent, currentFunction)
			}
		}
		return
	case bonsaipython.KindFunctionDefinition:
		nameNode := node.ChildByField(bonsaipython.FieldName)
		if nameNode == nil {
			return
		}
		name := string(nameNode.Text(source))
		symbolName := name
		if parent != "" {
			symbolName = parent + "." + name
		}
		qualified := unit.Module + "." + symbolName
		signature := "def " + name
		if params := node.ChildByField(bonsaipython.FieldParameters); params != nil {
			signature += string(params.Text(source))
		}
		unit.Symbols = append(unit.Symbols, model.Symbol{
			Name:       name,
			Qualified:  qualified,
			Kind:       "function",
			Signature:  signature,
			Doc:        docString(node, source),
			Parent:     parent,
			Visibility: visibility(name),
			LineStart:  int(node.StartPoint.Row + 1),
			LineEnd:    int(node.EndPoint.Row + 1),
		})
		if body := node.ChildByField(bonsaipython.FieldBody); body != nil {
			for call := range body.Find(bonsaipython.KindCall) {
				fn := call.ChildByField(bonsaipython.FieldFunction)
				if fn == nil {
					continue
				}
				target := simpleName(string(fn.Text(source)))
				if target != "" {
					unit.Calls = append(unit.Calls, model.Reference{From: qualified, Name: target})
				}
			}
		}
		return
	case bonsaipython.KindClassDefinition:
		nameNode := node.ChildByField(bonsaipython.FieldName)
		if nameNode == nil {
			return
		}
		name := string(nameNode.Text(source))
		symbolName := name
		if parent != "" {
			symbolName = parent + "." + name
		}
		qualified := unit.Module + "." + symbolName
		unit.Symbols = append(unit.Symbols, model.Symbol{
			Name:       name,
			Qualified:  qualified,
			Kind:       "class",
			Signature:  "class " + name,
			Doc:        docString(node, source),
			Parent:     parent,
			Visibility: visibility(name),
			LineStart:  int(node.StartPoint.Row + 1),
			LineEnd:    int(node.EndPoint.Row + 1),
		})
		if bases := node.ChildByField(bonsaipython.FieldSuperclasses); bases != nil {
			for _, base := range splitNames(string(bases.Text(source))) {
				unit.Bases = append(unit.Bases, model.Reference{From: qualified, Name: base})
			}
		}
		if body := node.ChildByField(bonsaipython.FieldBody); body != nil {
			for _, child := range body.Children {
				walk(child, source, unit, symbolName, "")
			}
		}
		return
	case bonsaipython.KindImportStatement, bonsaipython.KindImportFromStatement:
		for _, imported := range parseImports(string(node.Text(source))) {
			unit.Imports = append(unit.Imports, imported)
		}
		return
	}
	for _, child := range node.Children {
		walk(child, source, unit, parent, currentFunction)
	}
}

func parseImports(text string) []model.Import {
	text = strings.TrimSpace(text)
	switch {
	case strings.HasPrefix(text, "import "):
		text = strings.TrimPrefix(text, "import ")
		result := make([]model.Import, 0)
		for _, entry := range strings.Split(text, ",") {
			entry = strings.TrimSpace(entry)
			parts := strings.SplitN(entry, " as ", 2)
			imported := model.Import{Path: strings.TrimSpace(parts[0])}
			if len(parts) == 2 {
				imported.Alias = strings.TrimSpace(parts[1])
			}
			if imported.Path != "" {
				result = append(result, imported)
			}
		}
		return result
	case strings.HasPrefix(text, "from "):
		text = strings.TrimPrefix(text, "from ")
		parts := strings.SplitN(text, " import ", 2)
		if len(parts) != 2 {
			return nil
		}
		path := strings.TrimSpace(parts[0])
		if path == "" {
			return nil
		}
		result := []model.Import{{Path: path}}
		for _, name := range strings.Split(parts[1], ",") {
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			result = append(result, model.Import{Path: path, Alias: name})
		}
		return result
	}
	return nil
}

func splitNames(text string) []string {
	text = strings.TrimSpace(text)
	text = strings.TrimPrefix(text, "(")
	text = strings.TrimSuffix(text, ")")
	result := make([]string, 0)
	for _, entry := range strings.Split(text, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		result = append(result, simpleName(entry))
	}
	return result
}

func simpleName(text string) string {
	text = strings.TrimSpace(text)
	if index := strings.LastIndexByte(text, '.'); index >= 0 {
		text = text[index+1:]
	}
	text = strings.Trim(text, "()[]{} ")
	return text
}

func docString(node *bonsaipython.Node, source []byte) string {
	body := node.ChildByField(bonsaipython.FieldBody)
	if body == nil || len(body.Children) == 0 {
		return ""
	}
	first := body.Children[0]
	if first.Kind != "expression_statement" || len(first.Children) == 0 {
		return ""
	}
	value := first.Children[0]
	if value.Kind != "string" {
		return ""
	}
	return strings.Trim(strings.TrimSpace(string(value.Text(source))), `"'`)
}

func moduleName(path string) string {
	path = strings.TrimSuffix(strings.ReplaceAll(path, "\\", "/"), ".py")
	path = strings.TrimSuffix(path, "/__init__")
	return strings.ReplaceAll(path, "/", ".")
}

func visibility(name string) string {
	if name == "" || strings.HasPrefix(name, "_") {
		return "private"
	}
	return "public"
}
