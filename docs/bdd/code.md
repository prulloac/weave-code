---
type: behavior
title: "Weave Code Extension Behavior"
description: "Extension-side scenarios for Go and Python extraction, relation output and lexical fallback."
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
- **And** source extensions include Go and Python

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

## Feature: Compatibility

### Scenario: Static build

- **Given** the release build
- **When** I inspect the binary
- **Then** it is built with `CGO_ENABLED=0`
- **And** it does not require a language server or network
