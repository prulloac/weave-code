package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/prulloac/weave-code/internal/analyze"
	"github.com/prulloac/weave-code/internal/model"
)

var version = "dev"

const protocolVersion = "weave.cli/v1"

var sourceExtensions = []string{
	".go", ".py", ".js", ".jsx", ".ts", ".tsx", ".rs",
	".c", ".h", ".cc", ".cpp", ".cxx", ".hpp", ".hh", ".hxx",
	".java", ".scala", ".sc", ".dart", ".zig",
}

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "weave-code:", err)
		os.Exit(1)
	}
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	if len(args) > 0 {
		switch args[0] {
		case "--version", "-v", "version":
			fmt.Fprintf(stdout, "weave-code %s (weave.extension/v1)\n", version)
			return nil
		case "--describe":
			jsonMode := len(args) > 1 && args[1] == "--json"
			return describe(stdout, jsonMode)
		}
	}
	var request model.Request
	if err := json.NewDecoder(stdin).Decode(&request); err != nil {
		return fmt.Errorf("decode project analysis request: %w", err)
	}
	if request.V != protocolVersion || request.Type != "project_analysis_request" {
		return fmt.Errorf("unsupported project analysis request")
	}
	if symbolDocumentsEnabled(request.BundleRoot) {
		request.Options.SymbolDocuments = true
	}
	response, err := analyze.Run(request)
	if err != nil {
		return err
	}
	return json.NewEncoder(stdout).Encode(response)
}

func describe(stdout io.Writer, jsonMode bool) error {
	if jsonMode {
		return json.NewEncoder(stdout).Encode(map[string]any{
			"v":                 "weave.extension/v1",
			"name":              "code",
			"source_extensions": sourceExtensions,
			"analysis_modes":    []string{"project"},
		})
	}
	fmt.Fprintln(stdout, "name: code")
	fmt.Fprintln(stdout, "mode: project analysis")
	fmt.Fprintf(stdout, "source extensions: %v\n", sourceExtensions)
	return nil
}
