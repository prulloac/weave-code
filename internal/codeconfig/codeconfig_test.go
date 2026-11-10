package codeconfig

import (
	"os"
	"path/filepath"
	"testing"
)

func writeConfig(t *testing.T, root, body string) {
	t.Helper()
	path := filepath.Join(root, ".weave", "code.yaml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadConfig(t *testing.T) {
	root := t.TempDir()
	writeConfig(t, root, "# code settings\nsymbol_documents: true\nparse_generated: yes\nignore:\n  - \"third_party/**\"\n  - '**/*_test.go'\n")
	config := Load(root)
	if !config.SymbolDocuments {
		t.Fatal("expected symbol documents")
	}
	if !config.ParseGenerated {
		t.Fatal("expected parse_generated")
	}
	if len(config.Ignore) != 2 || config.Ignore[0] != "third_party/**" || config.Ignore[1] != "**/*_test.go" {
		t.Fatalf("ignore = %v", config.Ignore)
	}
}

func TestLoadInlineIgnore(t *testing.T) {
	root := t.TempDir()
	writeConfig(t, root, "ignore: [\"drafts/**\", \"*.min.js\"]\n")
	config := Load(root)
	if len(config.Ignore) != 2 || config.Ignore[0] != "drafts/**" || config.Ignore[1] != "*.min.js" {
		t.Fatalf("ignore = %v", config.Ignore)
	}
}

func TestLoadConfigFromEnvironment(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "custom.yaml")
	if err := os.WriteFile(path, []byte("symbol_documents: true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("WEAVE_CODE_CONFIG", path)
	if !Load(t.TempDir()).SymbolDocuments {
		t.Fatal("expected environment config")
	}
}

func TestSkipReason(t *testing.T) {
	config := Config{Ignore: []string{"third_party/**", "**/*_test.go", "*.md"}}
	cases := map[string]string{
		"src/main.go":            "",
		"vendor/lib.go":          "ignored directory",
		"build/app.js":           "ignored directory",
		"node_modules/pkg/a.js":  "ignored directory",
		".git/config":            "ignored directory",
		"app.gen.go":             "generated file",
		"app.generated.ts":       "generated file",
		"bundle.min.js":          "generated file",
		"app_test.go":            "ignored path",
		"src/app_test.go":        "ignored path",
		"third_party/lib.go":     "ignored path",
		"third_party/sub/lib.go": "ignored path",
		"notes.md":               "ignored path",
		"src/notes.md":           "ignored path",
		"src/main.go~":           "ignored file",
		"src/.hidden.go":         "ignored file",
		"src/swap.tmp":           "ignored file",
	}
	for path, want := range cases {
		if got := config.SkipReason(path); got != want {
			t.Fatalf("SkipReason(%q) = %q, want %q", path, got, want)
		}
	}
}

func TestParseGeneratedOptIn(t *testing.T) {
	config := Config{ParseGenerated: true}
	if reason := config.SkipReason("app.gen.go"); reason != "" {
		t.Fatalf("generated file should be included, got %q", reason)
	}
}
