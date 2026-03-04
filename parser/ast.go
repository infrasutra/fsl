package parser

// Schema represents a complete FSL schema
type Schema struct {
	Types []TypeDef `json:"types"`
	Enums []EnumDef `json:"enums,omitempty"` // Named enum definitions
}

// TypeDef represents a type definition
type TypeDef struct {
	Name       string      `json:"name"`
	Decorators []Decorator `json:"decorators,omitempty"`
	Fields     []FieldDef  `json:"fields"`
}

// EnumDef represents a named enum definition
type EnumDef struct {
	Name   string   `json:"name"`
	Values []string `json:"values"`
}

// FieldDef represents a field within a type
type FieldDef struct {
	Name       string         `json:"name"`
	Type       string         `json:"type"`
	Required   bool           `json:"required"`
	Array      bool           `json:"array"`
	ArrayReq   bool           `json:"arrayRequired,omitempty"` // Array itself is required
	Decorators map[string]any `json:"decorators,omitempty"`

	// Phase 2: Enum and relation support
	InlineEnum []string `json:"inlineEnum,omitempty"` // Inline enum values (e.g., "draft" | "published")
	IsRelation bool     `json:"isRelation,omitempty"` // True if field is a relation to another type
}

// Decorator represents a decorator application
type Decorator struct {
	Name string `json:"name"`
	Args []any  `json:"args,omitempty"`
}

// RelationInfo holds parsed relation decorator details
type RelationInfo struct {
	TargetType string `json:"targetType"`          // The type being referenced
	Inverse    string `json:"inverse,omitempty"`   // Inverse field name for bidirectional relations
	OnDelete   string `json:"onDelete,omitempty"`  // Cascade behavior: "cascade", "restrict", "setNull"
	IsSelfRef  bool   `json:"isSelfRef,omitempty"` // True if relation points to same type
}

// RichTextConfig holds parsed RichText decorator configuration
type RichTextConfig struct {
	AllowedBlocks []string `json:"allowedBlocks,omitempty"` // Allowed block types
}

// AssetConfig holds parsed Image/File decorator configuration
type AssetConfig struct {
	MaxSize int64    `json:"maxSize,omitempty"` // Max file size in bytes
	Formats []string `json:"formats,omitempty"` // Allowed formats
}
