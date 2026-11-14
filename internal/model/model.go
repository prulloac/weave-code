package model

// File is one source file offered by Weave.
type File struct {
	ID          string `json:"id"`
	Path        string `json:"path"`
	MediaType   string `json:"media_type,omitempty"`
	ContentHash string `json:"content_hash,omitempty"`
}

// Options controls project analysis.
type Options struct {
	SymbolDocuments bool     `json:"symbol_documents"`
	Relations       []string `json:"relations,omitempty"`
	ParseGenerated  bool     `json:"parse_generated,omitempty"`
	Ignore          []string `json:"ignore,omitempty"`
}

// Request is the project_analysis_request payload.
type Request struct {
	V          string  `json:"v"`
	Type       string  `json:"type"`
	BundleRoot string  `json:"bundle_root"`
	Files      []File  `json:"files"`
	Options    Options `json:"options"`
}

// Document is one file, symbol or external dependency.
type Document struct {
	ID              string         `json:"id"`
	Type            string         `json:"type,omitempty"`
	ContentKind     string         `json:"content_kind,omitempty"`
	Title           string         `json:"title,omitempty"`
	Description     string         `json:"description,omitempty"`
	Keywords        []string       `json:"keywords,omitempty"`
	Body            string         `json:"body,omitempty"`
	ContentMetadata map[string]any `json:"content_metadata,omitempty"`
	SymbolDocuments []Document     `json:"symbol_documents,omitempty"`
}

// Artifact is one extension-owned export artifact.
type Artifact struct {
	Path      string `json:"path"`
	MediaType string `json:"media_type,omitempty"`
	Content   any    `json:"content"`
}

// Relation is one code relationship.
type Relation struct {
	From string `json:"from"`
	To   string `json:"to"`
	Kind string `json:"kind"`
}

// Response is the project_analysis_response payload.
type Response struct {
	V                 string     `json:"v"`
	Documents         []Document `json:"documents"`
	ExternalDocuments []Document `json:"external_documents,omitempty"`
	Relations         []Relation `json:"relations,omitempty"`
	Artifacts         []Artifact `json:"artifacts,omitempty"`
	Warnings          []string   `json:"warnings,omitempty"`
}
