package golanglang

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"strconv"
	"strings"

	"github.com/prulloac/weave-code/internal/model"
)

// Parse parses one Go source file.
func Parse(file model.File, source []byte) (model.Unit, error) {
	fset := token.NewFileSet()
	parsed, err := parser.ParseFile(fset, file.Path, source, parser.ParseComments)
	if err != nil {
		return model.Unit{}, err
	}
	unit := model.Unit{
		File:     file,
		Language: "go",
		Package:  parsed.Name.Name,
		Parser:   "go/parser",
		Status:   "parsed",
	}
	if parsed.Doc != nil {
		unit.Doc = strings.TrimSpace(parsed.Doc.Text())
	}
	for _, spec := range parsed.Imports {
		path, _ := strconv.Unquote(spec.Path.Value)
		alias := ""
		if spec.Name != nil {
			alias = spec.Name.Name
		}
		unit.Imports = append(unit.Imports, model.Import{Path: path, Alias: alias})
	}
	qualified := func(name string) string {
		if unit.Package == "" {
			return name
		}
		return unit.Package + "." + name
	}
	for _, decl := range parsed.Decls {
		switch typed := decl.(type) {
		case *ast.FuncDecl:
			kind := "function"
			name := typed.Name.Name
			parent := ""
			if typed.Recv != nil && len(typed.Recv.List) > 0 {
				kind = "method"
				parent = receiverName(typed.Recv.List[0].Type)
				if parent != "" {
					name = parent + "." + typed.Name.Name
				}
			}
			signature := nodeString(fset, typed.Type)
			symbol := model.Symbol{
				Name:       typed.Name.Name,
				Qualified:  qualified(name),
				Kind:       kind,
				Signature:  signature,
				Doc:        commentText(typed.Doc),
				Parent:     parent,
				Visibility: visibility(typed.Name.Name),
				LineStart:  fset.Position(typed.Pos()).Line,
				LineEnd:    fset.Position(typed.End()).Line,
			}
			unit.Symbols = append(unit.Symbols, symbol)
			collectCalls(typed.Body, symbol.Qualified, &unit)
		case *ast.GenDecl:
			for _, spec := range typed.Specs {
				switch entry := spec.(type) {
				case *ast.TypeSpec:
					unit.Symbols = append(unit.Symbols, model.Symbol{
						Name:       entry.Name.Name,
						Qualified:  qualified(entry.Name.Name),
						Kind:       "type",
						Signature:  "type " + entry.Name.Name,
						Doc:        commentText(entry.Doc),
						Visibility: visibility(entry.Name.Name),
						LineStart:  fset.Position(entry.Pos()).Line,
						LineEnd:    fset.Position(entry.End()).Line,
					})
					collectBase(entry.Type, qualified(entry.Name.Name), &unit)
				case *ast.ValueSpec:
					kind := strings.ToLower(typed.Tok.String())
					for _, name := range entry.Names {
						unit.Symbols = append(unit.Symbols, model.Symbol{
							Name:       name.Name,
							Qualified:  qualified(name.Name),
							Kind:       kind,
							Signature:  kind + " " + name.Name,
							Doc:        commentText(entry.Doc),
							Visibility: visibility(name.Name),
							LineStart:  fset.Position(name.Pos()).Line,
							LineEnd:    fset.Position(name.End()).Line,
						})
					}
				}
			}
		}
	}
	return unit, nil
}

func collectCalls(body *ast.BlockStmt, from string, unit *model.Unit) {
	if body == nil {
		return
	}
	ast.Inspect(body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		name := callName(call.Fun)
		if name != "" {
			unit.Calls = append(unit.Calls, model.Reference{From: from, Name: name})
		}
		return true
	})
}

func callName(expr ast.Expr) string {
	switch typed := expr.(type) {
	case *ast.Ident:
		return typed.Name
	case *ast.SelectorExpr:
		return typed.Sel.Name
	}
	return ""
}

func collectBase(expr ast.Expr, from string, unit *model.Unit) {
	switch typed := expr.(type) {
	case *ast.StructType:
		for _, field := range typed.Fields.List {
			if len(field.Names) == 0 {
				if name := typeName(field.Type); name != "" {
					unit.Bases = append(unit.Bases, model.Reference{From: from, Name: name})
				}
			}
		}
	case *ast.InterfaceType:
		for _, field := range typed.Methods.List {
			if len(field.Names) == 0 {
				if name := typeName(field.Type); name != "" {
					unit.Bases = append(unit.Bases, model.Reference{From: from, Name: name})
				}
			}
		}
	}
}

func typeName(expr ast.Expr) string {
	switch typed := expr.(type) {
	case *ast.Ident:
		return typed.Name
	case *ast.StarExpr:
		return typeName(typed.X)
	case *ast.SelectorExpr:
		return typed.Sel.Name
	}
	return ""
}

func receiverName(expr ast.Expr) string {
	switch typed := expr.(type) {
	case *ast.StarExpr:
		return receiverName(typed.X)
	case *ast.Ident:
		return typed.Name
	}
	return ""
}

func nodeString(fset *token.FileSet, node any) string {
	var buffer bytes.Buffer
	if err := format.Node(&buffer, fset, node); err != nil {
		return ""
	}
	return buffer.String()
}

func commentText(group *ast.CommentGroup) string {
	if group == nil {
		return ""
	}
	return strings.TrimSpace(group.Text())
}

func visibility(name string) string {
	if ast.IsExported(name) {
		return "public"
	}
	return "private"
}
