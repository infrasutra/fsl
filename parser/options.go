package parser

import "fmt"

// Options describes the definitions a schema is validated and compiled against.
//
// ExternalTypes are content types defined in other schemas; fields may use them
// as relation targets. Library holds shared section types; @slices mappings may
// reference them, and they are compiled into the schema's components with
// Shared set. When RejectShadowing is true, a type defined in the schema with
// the same name as a library type is an error; otherwise the local type wins.
type Options struct {
	ExternalTypes   []string
	Library         *Schema
	RejectShadowing bool
}

// ParseWithDiagnosticsAndOptions parses and validates FSL against opts and
// returns diagnostics with line and column positions.
func ParseWithDiagnosticsAndOptions(source string, opts Options) *DiagnosticsResult {
	result := &DiagnosticsResult{Valid: true, Diagnostics: []Diagnostic{}}
	schema, err := NewParser(NewLexer(source)).ParseSchema()
	if err != nil {
		result.Valid = false
		result.Diagnostics = append(result.Diagnostics, parseErrorToDiagnostic(err.Error(), source))
		return result
	}
	result.Schema = schema
	for _, valErr := range NewValidatorWithOptions(schema, opts).Validate() {
		result.Valid = false
		result.Diagnostics = append(result.Diagnostics, validationErrorToDiagnostic(valErr, source))
	}
	return result
}

// ParseAndCompileWithOptions parses, validates and compiles FSL against opts.
func ParseAndCompileWithOptions(source, name, apiID string, singleton bool, opts Options) (*CompiledSchema, error) {
	schema, err := NewParser(NewLexer(source)).ParseSchema()
	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}
	if err := ValidateSchemaWithOptions(schema, opts); err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}
	compiled, err := CompileWithOptions(schema, name, apiID, singleton, opts)
	if err != nil {
		return nil, fmt.Errorf("compilation error: %w", err)
	}
	return compiled, nil
}

// ParseLibrary parses and validates a section library: a definition whose types
// are shared sections that other schemas reference from @slices.
func ParseLibrary(source string) (*Schema, error) {
	return Parse(source)
}

// CompileLibrary compiles every type of a section library as a shared component.
func CompileLibrary(library *Schema) ([]CompiledComponent, error) {
	targets := make(map[string]bool, len(library.Types))
	for _, t := range library.Types {
		targets[t.Name] = true
	}
	return compileSliceComponents(targets, map[string]*TypeDef{}, libraryIndex(library))
}

func libraryIndex(library *Schema) map[string]*TypeDef {
	index := map[string]*TypeDef{}
	if library == nil {
		return index
	}
	for i := range library.Types {
		index[library.Types[i].Name] = &library.Types[i]
	}
	return index
}
