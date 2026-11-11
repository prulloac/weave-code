# AGENTS.md

`weave-code` is the standalone code knowledge extension for Weave. It speaks `weave.cli/v1` project
analysis over stdin/stdout and never imports Weave engine internals.

## Commands

- Build: `go build -o weave-code .`
- All tests: `go test ./...`
- Race: `go test -race -count=1 -shuffle=on ./...`
- Vet / format: `go vet ./...`; `gofmt -l .` must print nothing
- Describe: `go run . --describe --json`
- Version: `go run . --version`
- Full gate: `./scripts/verify.sh`
- User-level install: `./scripts/install.sh --help` (or `scripts/install.ps1` on Windows)

## Layout

```text
main.go                 CLI, describe/version, project analysis entrypoint
internal/model/         request, response, unit and symbol types
internal/analyze/       project orchestration and relation resolution
internal/golanglang/     Go standard-library parser
internal/pythonlang/     pure-Go tree-sitter Python parser
internal/treesitterlang/ pure-Go tree-sitter parsers for the remaining languages
internal/codeconfig/     `.weave/code.yaml` loading and ignore matching
internal/lexical/        fallback extraction
docs/bdd/               extension-side behavior scenarios
scripts/                verification and installers
```

## Invariants

1. **Static builds:** no C libraries, no compiler, no language server, no network.
2. **Contract only:** speak `weave.cli/v1`; never import Weave packages or read a database.
3. **Source only:** read files under `bundle_root`/`path`; never write the bundle.
4. **Bounded:** enforce the file, project, keyword, symbol and response limits from the specification.
5. **Deterministic:** sort relations and keep document order.
6. **Best effort:** a parse failure falls back to lexical metadata and one warning.
7. **No execution:** never run source code or build tools.
8. **User-level install:** installers never require administrator rights.

## Testing quirks

- Go extraction uses the standard library parser.
- Python extraction uses `github.com/msuozzo/bonsai/bonsai-python`, a pure-Go tree-sitter runtime.
- JavaScript, TypeScript, Rust, C, C++, Java, Scala, Dart and Zig use
  `github.com/odvcencio/gotreesitter` grammar packages. Dart method bodies are siblings of their
  signatures, so symbol context carries across sibling nodes.
- Fixtures are temporary directories; no real project or database is used.
- Keep `--describe --json` stable; Weave reads it before every project write.

## Documentation split

- This repository owns usage docs and extension-side behavior.
- The behavioral contract lives in
  [`weave-docs/specs/weave-code.md`](https://github.com/prulloac/weave-docs/blob/main/specs/weave-code.md).
- Core ingestion, graph and relation behavior live in the Weave repository.

## Release

- Tags use `vX.Y.Z`; the release workflow builds six static archives and a draft GitHub Release.
- Keep the binary name `weave-code`; Weave discovers extensions by the `weave-` prefix.
- Current product release: `v0.8.0`. It requires `weave v0.8.0` project analysis.
