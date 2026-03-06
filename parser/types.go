package parser

// Built-in types supported in Flux CMS
const (
	// Phase 1 types
	TypeString   = "String"
	TypeText     = "Text"
	TypeInt      = "Int"
	TypeFloat    = "Float"
	TypeBoolean  = "Boolean"
	TypeDateTime = "DateTime"
	TypeJSON     = "JSON"

	// Phase 2 types
	TypeDate     = "Date"     // Date-only (no time)
	TypeRichText = "RichText" // Formatted content with block types
	TypeImage    = "Image"    // Image asset reference
	TypeFile     = "File"     // Generic file reference
)

var BuiltinTypes = map[string]bool{
	// Phase 1
	TypeString:   true,
	TypeText:     true,
	TypeInt:      true,
	TypeFloat:    true,
	TypeBoolean:  true,
	TypeDateTime: true,
	TypeJSON:     true,
	// Phase 2
	TypeDate:     true,
	TypeRichText: true,
	TypeImage:    true,
	TypeFile:     true,
}

// Built-in decorators
const (
	// Phase 1 field decorators
	DecRequired   = "required"   // implicit with !
	DecDefault    = "default"    // default value
	DecMaxLength  = "maxLength"  // max string length
	DecMinLength  = "minLength"  // min string length
	DecMin        = "min"        // min numeric value
	DecMax        = "max"        // max numeric value
	DecPattern    = "pattern"    // regex pattern
	DecUnique     = "unique"     // uniqueness constraint
	DecIndex      = "index"      // index hint
	DecSearchable = "searchable" // search hint

	// Phase 2 field decorators
	DecRelation  = "relation"  // relation to another type
	DecSlices = "slices" // typed slice zone for dynamic page sections
	DecMaxSize   = "maxSize"   // max file size in bytes
	DecFormats   = "formats"   // allowed file formats
	DecPrecision = "precision" // decimal precision for Float
	DecMinItems  = "minItems"  // min array length
	DecMaxItems  = "maxItems"  // max array length
	DecHidden    = "hidden"    // hide from API

	// Phase 2 type-level decorators
	DecCollection = "collection" // custom collection name
	DecSingleton  = "singleton"  // singleton type (one document)

	// Display decorators (field-level) - for UI customization
	DecLabel       = "label"       // custom display label
	DecHelp        = "help"        // help text / tooltip
	DecPlaceholder = "placeholder" // input placeholder text

	// Schema metadata decorators (type-level)
	DecIcon        = "icon"        // Lucide icon name for schema
	DecDescription = "description" // schema description in FSL
)

// Reserved field names that cannot be used in schemas
var ReservedFieldNames = map[string]bool{
	"id":        true,
	"createdAt": true,
	"updatedAt": true,
}

// ValidImageFormats defines allowed image file formats
var ValidImageFormats = map[string]bool{
	"jpg":  true,
	"jpeg": true,
	"png":  true,
	"gif":  true,
	"webp": true,
	"svg":  true,
	"avif": true,
}

// ValidFileFormats defines commonly allowed file formats
var ValidFileFormats = map[string]bool{
	// Images
	"jpg": true, "jpeg": true, "png": true, "gif": true, "webp": true, "svg": true, "avif": true,
	// Documents
	"pdf": true, "doc": true, "docx": true, "xls": true, "xlsx": true, "ppt": true, "pptx": true,
	// Archives
	"zip": true, "tar": true, "gz": true,
	// Text
	"txt": true, "csv": true, "json": true, "xml": true,
	// Media
	"mp3": true, "mp4": true, "wav": true, "avi": true, "mov": true,
}
