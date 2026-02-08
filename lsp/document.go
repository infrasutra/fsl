package lsp

import (
	"strings"
	"sync"

	"github.com/infrasutra/fsl/parser"
)

// Document represents an open text document
type Document struct {
	URI     string
	Content string
	Version int
	Lines   []string

	// Parsed data (cached)
	mu           sync.RWMutex
	parsedSchema *parser.Schema
	parseResult  *parser.DiagnosticsResult
}

// NewDocument creates a new document
func NewDocument(uri, content string, version int) *Document {
	d := &Document{
		URI:     uri,
		Content: content,
		Version: version,
	}
	d.updateLines()
	d.parse()
	return d
}

// Update updates the document content
func (d *Document) Update(content string, version int) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.Content = content
	d.Version = version
	d.updateLines()
	d.parseUnsafe()
}

func (d *Document) updateLines() {
	d.Lines = strings.Split(d.Content, "\n")
}

func (d *Document) parse() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.parseUnsafe()
}

func (d *Document) parseUnsafe() {
	result := parser.ParseWithDiagnostics(d.Content)
	d.parseResult = result
	if result.Schema != nil {
		d.parsedSchema = result.Schema
	}
}

// GetSchema returns the parsed schema
func (d *Document) GetSchema() *parser.Schema {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.parsedSchema
}

// GetParseResult returns the parse result with diagnostics
func (d *Document) GetParseResult() *parser.DiagnosticsResult {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.parseResult
}

// GetLine returns a specific line (0-indexed)
func (d *Document) GetLine(line int) string {
	if line < 0 || line >= len(d.Lines) {
		return ""
	}
	return d.Lines[line]
}

// GetWordAt returns the word at the given position
func (d *Document) GetWordAt(pos Position) string {
	line := d.GetLine(pos.Line)
	if line == "" {
		return ""
	}

	// Find word boundaries
	start := pos.Character
	end := pos.Character

	// Go backwards to find start
	for start > 0 && isWordChar(line[start-1]) {
		start--
	}

	// Go forwards to find end
	for end < len(line) && isWordChar(line[end]) {
		end++
	}

	if start >= end || start >= len(line) {
		return ""
	}

	return line[start:end]
}

// GetWordRange returns the range of the word at the given position
func (d *Document) GetWordRange(pos Position) Range {
	line := d.GetLine(pos.Line)
	if line == "" {
		return Range{Start: pos, End: pos}
	}

	start := pos.Character
	end := pos.Character

	for start > 0 && isWordChar(line[start-1]) {
		start--
	}

	for end < len(line) && isWordChar(line[end]) {
		end++
	}

	return Range{
		Start: Position{Line: pos.Line, Character: start},
		End:   Position{Line: pos.Line, Character: end},
	}
}

// OffsetToPosition converts a byte offset to a position
func (d *Document) OffsetToPosition(offset int) Position {
	line := 0
	col := 0
	for i := 0; i < offset && i < len(d.Content); i++ {
		if d.Content[i] == '\n' {
			line++
			col = 0
		} else {
			col++
		}
	}
	return Position{Line: line, Character: col}
}

// PositionToOffset converts a position to a byte offset
func (d *Document) PositionToOffset(pos Position) int {
	offset := 0
	for i := 0; i < pos.Line && i < len(d.Lines); i++ {
		offset += len(d.Lines[i]) + 1 // +1 for newline
	}
	offset += pos.Character
	return offset
}

func isWordChar(c byte) bool {
	return (c >= 'a' && c <= 'z') ||
		(c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') ||
		c == '_'
}

// DocumentStore manages open documents
type DocumentStore struct {
	mu        sync.RWMutex
	documents map[string]*Document
}

// NewDocumentStore creates a new document store
func NewDocumentStore() *DocumentStore {
	return &DocumentStore{
		documents: make(map[string]*Document),
	}
}

// Open opens a new document
func (s *DocumentStore) Open(uri, content string, version int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.documents[uri] = NewDocument(uri, content, version)
}

// Update updates an existing document
func (s *DocumentStore) Update(uri, content string, version int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if doc, ok := s.documents[uri]; ok {
		doc.Update(content, version)
	} else {
		s.documents[uri] = NewDocument(uri, content, version)
	}
}

// Close closes a document
func (s *DocumentStore) Close(uri string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.documents, uri)
}

// Get retrieves a document by URI
func (s *DocumentStore) Get(uri string) *Document {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.documents[uri]
}

// All returns all open documents
func (s *DocumentStore) All() []*Document {
	s.mu.RLock()
	defer s.mu.RUnlock()

	docs := make([]*Document, 0, len(s.documents))
	for _, doc := range s.documents {
		docs = append(docs, doc)
	}
	return docs
}
