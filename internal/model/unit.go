package model

// Symbol is one declaration in a source file.
type Symbol struct {
	Name       string `json:"name"`
	Qualified  string `json:"qualified"`
	Kind       string `json:"kind"`
	Signature  string `json:"signature,omitempty"`
	Doc        string `json:"doc,omitempty"`
	Parent     string `json:"parent,omitempty"`
	Visibility string `json:"visibility,omitempty"`
	LineStart  int    `json:"line_start,omitempty"`
	LineEnd    int    `json:"line_end,omitempty"`
}

// Import is one imported module or package.
type Import struct {
	Path  string `json:"path"`
	Alias string `json:"alias,omitempty"`
}

// Reference is an unresolved relationship from one local symbol.
type Reference struct {
	From string `json:"from"`
	Name string `json:"name"`
}

// Unit is one parsed source file.
type Unit struct {
	File       File
	Language   string
	Module     string
	Package    string
	Doc        string
	Parser     string
	Status     string
	Symbols    []Symbol
	Imports    []Import
	Calls      []Reference
	References []Reference
	Bases      []Reference
	Warnings   []string
}
