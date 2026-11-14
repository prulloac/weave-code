---
type: behavior
title: "Weave Code Extension Behavior"
description: "Extension-side scenarios for Go, Python and tree-sitter extraction, relation output and lexical fallback."
status: draft
---

# Behavior Scenarios: Weave Code Extension

Core contract: [weave-code.md](https://github.com/prulloac/weave-docs/blob/main/specs/weave-code.md)

## Feature: Manifest

### Scenario: The extension declares project analysis

- **Given** the built `weave-code` binary
- **When** I run `weave-code --describe --json`
- **Then** it returns `weave.extension/v1`
- **And** the name is `code`
- **And** analysis modes include `project`
- **And** source extensions include Go, Python, JavaScript, TypeScript, Rust, C, C++, Java, Scala,
  Dart and Zig

## Feature: Go extraction

### Scenario: Functions and types become symbols

- **Given** a Go file with functions and types
- **When** the project is analyzed
- **Then** each declaration appears as a symbol document when enabled
- **And** the file metadata names the package and parser

### Scenario: Calls and embedding resolve

- **Given** a Go function that calls another function
- **And** a struct that embeds another struct
- **When** the project is analyzed
- **Then** a `calls` relation links the functions
- **And** an `inherits` relation links the structs

## Feature: Python extraction

### Scenario: Classes, methods and calls resolve

- **Given** a Python module with classes, methods and a function call
- **When** the project is analyzed
- **Then** symbol documents use `<file>#<symbol>`
- **And** inheritance and call relations resolve

## Feature: Tree-sitter extraction

### Scenario: Remaining languages parse without C libraries

- **Given** a JavaScript, TypeScript, Rust, C, C++, Java, Scala, Dart or Zig source file
- **When** the project is analyzed
- **Then** symbol documents use `<file>#<symbol>` and nested methods use `<type>.<method>`
- **And** call, inheritance and implementation relations resolve in one project pass
- **And** the parser status is `parsed`, not `fallback`
- **And** the build contains no C library or CGO dependency

### Scenario: Calls and references resolve

- **Given** a function that calls another function
- **And** a function that references a type without calling it
- **When** the project is analyzed
- **Then** `calls` and `references` relations link the symbols

## Feature: Ignored paths and limits

### Scenario: Generated and ignored files are skipped

- **Given** `.weave/code.yaml` with an `ignore` pattern
- **And** a generated file, a vendor file and a matching path
- **When** the project is analyzed
- **Then** those files are not returned as documents
- **And** each skipped file produces a warning

### Scenario: Generated files can be parsed

- **Given** `.weave/code.yaml` with `parse_generated: true`
- **When** a generated source file is analyzed
- **Then** its document and symbols are returned

### Scenario: Oversized or invalid files are skipped

- **Given** a source file larger than 4 MiB or containing invalid UTF-8
- **When** the project is analyzed
- **Then** the file is skipped with a warning
- **And** symbol documents are capped at 2,000 per file

## Feature: Fallback and external targets

### Scenario: Invalid Go falls back

- **Given** a Go file with invalid syntax
- **When** the project is analyzed
- **Then** the file remains a document
- **And** its parser status is `fallback`
- **And** one warning is returned

### Scenario: Repeated external imports become documents

- **Given** two files importing the same external package
- **When** the project is analyzed
- **Then** one `_code/external/...` document is returned
- **And** both imports point to it

## Feature: Generic output

### Scenario: Documents declare their types

- **Given** a project with optional symbol documents and repeated external imports
- **When** the analyzer responds
- **Then** file documents declare `Code` / `blob`
- **And** symbol documents declare `CodeSymbol` / `symbol`
- **And** external documents declare `CodeExternal` / `symbol`

### Scenario: Code export is emitted as artifacts

- **Given** a project with symbols and relations
- **When** the analyzer responds
- **Then** it emits `_code/symbols.json` and `_code/relations.json` artifacts
- **And** core writes those artifacts instead of regenerating the legacy export

## Feature: Compatibility

### Scenario: Static build

- **Given** the release build
- **When** I inspect the binary
- **Then** it is built with `CGO_ENABLED=0`
- **And** it does not require a language server or network
