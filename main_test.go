package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/prulloac/weave-code/internal/model"
)

func writeFiles(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for name, body := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func analyzeProject(t *testing.T, root string, files []string, symbols bool) model.Response {
	t.Helper()
	request := model.Request{
		V: "weave.cli/v1", Type: "project_analysis_request", BundleRoot: root,
		Options: model.Options{SymbolDocuments: symbols},
	}
	for _, file := range files {
		request.Files = append(request.Files, model.File{ID: file, Path: file})
	}
	payload, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if err := run(nil, bytes.NewReader(payload), &stdout, &stderr); err != nil {
		t.Fatalf("run: %v stderr=%s", err, stderr.String())
	}
	var response model.Response
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	return response
}

func TestDescribeJSON(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := run([]string{"--describe", "--json"}, strings.NewReader(""), &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		V                string   `json:"v"`
		Name             string   `json:"name"`
		SourceExtensions []string `json:"source_extensions"`
		AnalysisModes    []string `json:"analysis_modes"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.V != "weave.extension/v1" || manifest.Name != "code" ||
		len(manifest.SourceExtensions) == 0 || len(manifest.AnalysisModes) != 1 || manifest.AnalysisModes[0] != "project" {
		t.Fatalf("unexpected manifest: %+v", manifest)
	}
}

func TestAnalyzeGo(t *testing.T) {
	root := writeFiles(t, map[string]string{
		"a.go": "package main\n\nimport \"fmt\"\n\ntype Base struct{}\ntype Server struct{ Base }\n\nfunc main() { Run(); fmt.Println(\"ok\") }\nfunc Run() {}\n",
		"b.go": "package main\n\nimport \"fmt\"\n\nfunc Helper() { fmt.Println(\"ok\") }\n",
	})
	response := analyzeProject(t, root, []string{"a.go", "b.go"}, true)
	if len(response.Documents) != 2 {
		t.Fatalf("expected 2 documents, got %d", len(response.Documents))
	}
	if len(response.Documents[0].SymbolDocuments) == 0 {
		t.Fatal("expected symbol documents for a.go")
	}
	assertRelation(t, response.Relations, "a.go#main", "a.go#Run", "calls")
	assertRelation(t, response.Relations, "a.go#Server", "a.go#Base", "inherits")
	assertRelation(t, response.Relations, "a.go", "_code/external/go/fmt", "imports")
	foundExternal := false
	for _, document := range response.ExternalDocuments {
		if document.ID == "_code/external/go/fmt" {
			foundExternal = true
		}
	}
	if !foundExternal {
		t.Fatalf("expected repeated external import document, got %+v", response.ExternalDocuments)
	}
}

func TestAnalyzePython(t *testing.T) {
	root := writeFiles(t, map[string]string{
		"pkg/mod.py": "import os\n\nclass Base:\n    pass\n\nclass Server(Base):\n    def run(self):\n        helper()\n\ndef helper():\n    pass\n",
	})
	response := analyzeProject(t, root, []string{"pkg/mod.py"}, true)
	if len(response.Documents) != 1 {
		t.Fatalf("expected one document, got %d", len(response.Documents))
	}
	assertRelation(t, response.Relations, "pkg/mod.py#Server", "pkg/mod.py#Base", "inherits")
	assertRelation(t, response.Relations, "pkg/mod.py#Server.run", "pkg/mod.py#helper", "calls")
}

func TestSymbolDocumentsFromEnvironment(t *testing.T) {
	root := writeFiles(t, map[string]string{
		"a.go": "package main\n\nfunc main() {}\n",
	})
	t.Setenv("WEAVE_CODE_SYMBOL_DOCUMENTS", "1")
	response := analyzeProject(t, root, []string{"a.go"}, false)
	if len(response.Documents) != 1 || len(response.Documents[0].SymbolDocuments) == 0 {
		t.Fatalf("expected symbol documents from environment, got %+v", response.Documents)
	}
}

func TestLexicalFallback(t *testing.T) {
	root := writeFiles(t, map[string]string{
		"broken.go": "package main\n\nfunc broken( {\n",
	})
	response := analyzeProject(t, root, []string{"broken.go"}, false)
	if len(response.Documents) != 1 {
		t.Fatalf("expected one document, got %d", len(response.Documents))
	}
	if response.Documents[0].ContentMetadata["status"] != "fallback" {
		t.Fatalf("expected fallback status, got %+v", response.Documents[0].ContentMetadata)
	}
}

func assertRelation(t *testing.T, relations []model.Relation, from, to, kind string) {
	t.Helper()
	for _, relation := range relations {
		if relation.From == from && relation.To == to && relation.Kind == kind {
			return
		}
	}
	t.Fatalf("missing relation %s -> %s (%s) in %+v", from, to, kind, relations)
}
