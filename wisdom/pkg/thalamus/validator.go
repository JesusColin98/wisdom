package thalamus

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xeipuuv/gojsonschema"
)

// Validator handles JSON schema validation for tool calls.
type Validator struct {
	schemas map[string]*gojsonschema.Schema
}

// NewValidator creates a new JSON schema validator.
func NewValidator() *Validator {
	return &Validator{
		schemas: make(map[string]*gojsonschema.Schema),
	}
}

// RegisterSchema registers a JSON schema for a specific tool or method.
func (v *Validator) RegisterSchema(name string, schemaJSON string) error {
	loader := gojsonschema.NewStringLoader(schemaJSON)
	schema, err := gojsonschema.NewSchema(loader)
	if err != nil {
		return fmt.Errorf("failed to parse schema for %s: %w", name, err)
	}
	v.schemas[name] = schema
	return nil
}

// LoadSchemasFromDir loads all .json files from a directory as schemas.
// The filename (without extension) is used as the method name.
func (v *Validator) LoadSchemasFromDir(dirPath string) error {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return fmt.Errorf("failed to read schema directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		path := filepath.Join(dirPath, entry.Name())
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read schema file %s: %w", entry.Name(), err)
		}

		name := strings.TrimSuffix(entry.Name(), ".json")
		if err := v.RegisterSchema(name, string(content)); err != nil {
			return fmt.Errorf("failed to register schema from %s: %w", entry.Name(), err)
		}
	}
	return nil
}

// Validate checks a JSON payload against a registered schema.
func (v *Validator) Validate(name string, payloadJSON string) error {
	schema, ok := v.schemas[name]
	if !ok {
		return fmt.Errorf("no schema registered for method: %s", name)
	}

	documentLoader := gojsonschema.NewStringLoader(payloadJSON)
	result, err := schema.Validate(documentLoader)
	if err != nil {
		return fmt.Errorf("validation execution failed: %w", err)
	}

	if !result.Valid() {
		var errs string
		for _, desc := range result.Errors() {
			errs += fmt.Sprintf("- %s\n", desc)
		}
		return fmt.Errorf("invalid parameters for %s:\n%s", name, errs)
	}

	return nil
}
