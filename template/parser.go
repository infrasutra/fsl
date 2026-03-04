package template

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/infrasutra/fsl/parser"
	"gopkg.in/yaml.v3"
)

// ParseFile parses a template file from disk
func ParseFile(path string) (*TemplateFile, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(path))
	var format string
	switch ext {
	case ".yaml", ".yml":
		format = "yaml"
	case ".json":
		format = "json"
	case ".fsl":
		format = "fsl"
	default:
		// Auto-detect from content
		format = ""
	}

	return ParseContent(string(content), format)
}

// ParseContent parses template content with optional format hint
func ParseContent(content, format string) (*TemplateFile, error) {
	var templateFile TemplateFile

	// Auto-detect format if not specified
	if format == "" {
		content = strings.TrimSpace(content)
		if strings.HasPrefix(content, "{") {
			format = "json"
		} else if strings.HasPrefix(content, "---") {
			format = "fsl"
		} else {
			format = "yaml"
		}
	}

	switch format {
	case "json":
		if err := json.Unmarshal([]byte(content), &templateFile); err != nil {
			return nil, fmt.Errorf("invalid JSON: %w", err)
		}
	case "yaml":
		if err := yaml.Unmarshal([]byte(content), &templateFile); err != nil {
			return nil, fmt.Errorf("invalid YAML: %w", err)
		}
	case "fsl":
		// FSL with YAML frontmatter
		parts := strings.SplitN(content, "---", 3)
		if len(parts) < 3 {
			return nil, fmt.Errorf("FSL file must have YAML frontmatter between --- markers")
		}
		// Parse frontmatter
		if err := yaml.Unmarshal([]byte(parts[1]), &templateFile); err != nil {
			return nil, fmt.Errorf("invalid YAML frontmatter: %w", err)
		}
		// FSL content
		templateFile.FSL = strings.TrimSpace(parts[2])
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}

	// Validate required fields
	if err := Validate(&templateFile); err != nil {
		return nil, err
	}

	return &templateFile, nil
}

// Validate checks that a template file has all required fields and valid FSL
func Validate(t *TemplateFile) error {
	if t.Name == "" {
		return fmt.Errorf("template name is required")
	}
	if t.FSL == "" {
		return fmt.Errorf("template FSL is required")
	}

	// Validate category if provided
	if t.Category != "" && !IsValidCategory(t.Category) {
		return fmt.Errorf("invalid category '%s', must be one of: content, commerce, marketing, system, custom", t.Category)
	}

	// Validate FSL syntax
	result := parser.ParseWithDiagnostics(t.FSL)
	if !result.Valid {
		var errors []string
		for _, diag := range result.Diagnostics {
			if diag.Severity == 1 { // Error severity
				errors = append(errors, fmt.Sprintf("line %d: %s", diag.StartLine, diag.Message))
			}
		}
		if len(errors) > 0 {
			return fmt.Errorf("FSL validation failed:\n%s", strings.Join(errors, "\n"))
		}
	}

	return nil
}

// ToYAML converts a template to YAML format
func ToYAML(t *TemplateFile) (string, error) {
	data, err := yaml.Marshal(t)
	if err != nil {
		return "", fmt.Errorf("failed to marshal YAML: %w", err)
	}
	return string(data), nil
}

// ToJSON converts a template to JSON format
func ToJSON(t *TemplateFile) (string, error) {
	data, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}
	return string(data), nil
}

// ToFSL converts a template to FSL format with YAML frontmatter
func ToFSL(t *TemplateFile) (string, error) {
	// Create frontmatter (without FSL field)
	frontmatter := struct {
		Name        string   `yaml:"name"`
		Description string   `yaml:"description,omitempty"`
		Icon        string   `yaml:"icon,omitempty"`
		Category    string   `yaml:"category,omitempty"`
		IsSingleton bool     `yaml:"is_singleton,omitempty"`
		Tags        []string `yaml:"tags,omitempty"`
	}{
		Name:        t.Name,
		Description: t.Description,
		Icon:        t.Icon,
		Category:    t.Category,
		IsSingleton: t.IsSingleton,
		Tags:        t.Tags,
	}

	data, err := yaml.Marshal(frontmatter)
	if err != nil {
		return "", fmt.Errorf("failed to marshal frontmatter: %w", err)
	}

	return fmt.Sprintf("---\n%s---\n%s", string(data), t.FSL), nil
}

// WriteFile writes a template to a file in the specified format
func WriteFile(t *TemplateFile, path, format string) error {
	var content string
	var err error

	// Determine format from extension if not specified
	if format == "" {
		ext := strings.ToLower(filepath.Ext(path))
		switch ext {
		case ".json":
			format = "json"
		case ".fsl":
			format = "fsl"
		default:
			format = "yaml"
		}
	}

	switch format {
	case "json":
		content, err = ToJSON(t)
	case "fsl":
		content, err = ToFSL(t)
	default:
		content, err = ToYAML(t)
	}

	if err != nil {
		return err
	}

	return os.WriteFile(path, []byte(content), 0o644)
}

// GenerateSlug creates a URL-friendly slug from a name
func GenerateSlug(name string) string {
	slug := strings.ToLower(name)
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ReplaceAll(slug, "_", "-")

	// Remove non-alphanumeric characters except hyphens
	result := make([]byte, 0, len(slug))
	for i := 0; i < len(slug); i++ {
		c := slug[i]
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' {
			result = append(result, c)
		}
	}

	// Remove consecutive hyphens
	finalSlug := string(result)
	for strings.Contains(finalSlug, "--") {
		finalSlug = strings.ReplaceAll(finalSlug, "--", "-")
	}

	// Trim leading/trailing hyphens
	finalSlug = strings.Trim(finalSlug, "-")

	// Ensure minimum length
	if len(finalSlug) < 3 {
		finalSlug = finalSlug + "-template"
	}

	return finalSlug
}
