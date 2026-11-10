# Weave Code Extension

`weave-code` is the first-party code knowledge extension for Weave. It parses source files, extracts
symbols, imports, calls, references and inheritance, and returns a project-wide analysis to the
Weave core.

```sh
weave ingest ~/repo --with-enricher code --db ~/.weave/index.db --json
weave list ~/repo --type Code --json
weave search ~/repo "ParseProject" --json
weave relate ~/repo src/main.go#main --depth 2 --relation imports,declares,calls --json
```

## Supported languages

- Full syntax-tree extraction: Go through the Go standard library parser; Python through a pure-Go
  tree-sitter runtime.
- Lexical fallback: JavaScript, TypeScript, Rust, C, C++, Java, Scala, Dart and Zig.
- Static, C-free builds only. No compiler, package manager, language server or network access.

## Discovered files

The extension declares these source extensions to Weave:

```text
.go .py .js .jsx .ts .tsx .rs
.c .h .cc .cpp .cxx .hpp .hh .hxx
.java .scala .sc .dart .zig
```

Weave scans them only when `--with-enricher code` is present.

## Output

For each source file the extension returns:

- title, description, keywords and content metadata;
- optional symbol documents when `WEAVE_CODE_SYMBOL_DOCUMENTS=1` or `.weave/code.yaml` enables them;
- imports, declares, references, calls, inherits and implements relations;
- external dependency documents once a module is referenced more than once.

## Install

```sh
curl -fsSL https://raw.githubusercontent.com/prulloac/weave-code/main/scripts/install.sh | sh
```

Windows PowerShell:

```powershell
irm https://raw.githubusercontent.com/prulloac/weave-code/main/scripts/install.ps1 | iex
```

Installers default to a user-writable directory and never require administrator rights.

## Develop

```sh
go build -o weave-code .
go test ./...
go vet ./...
./scripts/verify.sh
```

The project contract is
[`weave-code.md`](https://github.com/prulloac/weave-docs/blob/main/specs/weave-code.md).

## Issues

This repository does not accept issues. Report first-party defects in
[`prulloac/weave`](https://github.com/prulloac/weave/issues/new/choose).

## License

MIT.
