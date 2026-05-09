package thalamus_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/wisdom/pkg/thalamus"
)

func TestValidator(t *testing.T) {
	v := thalamus.NewValidator()

	// 1. Register a schema for a hypothetical tool
	schema := `{
		"type": "object",
		"properties": {
			"node_id": {"type": "string"},
			"depth": {"type": "integer", "minimum": 1}
		},
		"required": ["node_id"]
	}`
	if err := v.RegisterSchema("get_neighbors", schema); err != nil {
		t.Fatalf("failed to register schema: %v", err)
	}

	// 2. Valid payload
	validPayload := `{"node_id": "uuid-123", "depth": 2}`
	if err := v.Validate("get_neighbors", validPayload); err != nil {
		t.Errorf("expected valid payload to pass, got: %v", err)
	}

	// 3. Invalid payload: Missing required field
	invalidMissing := `{}`
	err := v.Validate("get_neighbors", invalidMissing)
	if err == nil {
		t.Error("expected error for missing required field, got nil")
	} else {
		t.Logf("Caught expected error (missing field): %v", err)
	}

	// 4. Invalid payload: Type mismatch (string instead of integer)
	invalidType := `{"node_id": "uuid-123", "depth": "high"}`
	err = v.Validate("get_neighbors", invalidType)
	if err == nil {
		t.Error("expected error for type mismatch, got nil")
	} else {
		t.Logf("Caught expected error (type mismatch): %v", err)
	}

	// 5. Invalid payload: Range violation
	invalidRange := `{"node_id": "uuid-123", "depth": 0}`
	err = v.Validate("get_neighbors", invalidRange)
	if err == nil {
		t.Error("expected error for range violation, got nil")
	} else {
		t.Logf("Caught expected error (range violation): %v", err)
	}
}

func TestLoadSchemasFromDir(t *testing.T) {
	// Create a temporary directory for schemas
	tmpDir, err := os.MkdirTemp("", "schemas")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	schemaJSON := `{
		"type": "object",
		"properties": {
			"id": {"type": "string"}
		},
		"required": ["id"]
	}`
	err = os.WriteFile(filepath.Join(tmpDir, "test_method.json"), []byte(schemaJSON), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Add a non-json file and a directory to test skips
	os.WriteFile(filepath.Join(tmpDir, "ignore_me.txt"), []byte("data"), 0644)
	os.Mkdir(filepath.Join(tmpDir, "some_dir"), 0755)

	v := thalamus.NewValidator()
	if err := v.LoadSchemasFromDir(tmpDir); err != nil {
		t.Fatalf("failed to load schemas from dir: %v", err)
	}

	// Verify the schema was registered
	err = v.Validate("test_method", `{"id": "valid"}`)
	if err != nil {
		t.Errorf("expected validation to pass, got: %v", err)
	}

	err = v.Validate("test_method", `{}`)
	if err == nil {
		t.Error("expected validation to fail for empty payload")
	}
}

func TestValidatorErrors(t *testing.T) {
	v := thalamus.NewValidator()

	t.Run("Register invalid schema", func(t *testing.T) {
		err := v.RegisterSchema("bad", `{"type": "invalid"}`)
		if err == nil {
			t.Error("expected error for invalid schema type")
		}
	})

	t.Run("Validate missing schema", func(t *testing.T) {
		err := v.Validate("missing", `{}`)
		if err == nil {
			t.Error("expected error for missing schema")
		}
	})

	t.Run("Validate malformed payload", func(t *testing.T) {
		v.RegisterSchema("test", `{"type": "object"}`)
		err := v.Validate("test", `{invalid json}`)
		if err == nil {
			t.Error("expected error for malformed payload")
		}
	})

	t.Run("LoadSchemasFromDir non-existent", func(t *testing.T) {
		err := v.LoadSchemasFromDir("/non/existent/path")
		if err == nil {
			t.Error("expected error for non-existent directory")
		}
	})

	t.Run("LoadSchemasFromDir invalid schema file", func(t *testing.T) {
		tmpDir, _ := os.MkdirTemp("", "bad-schemas")
		defer os.RemoveAll(tmpDir)
		os.WriteFile(filepath.Join(tmpDir, "bad.json"), []byte(`{"type": "invalid"}`), 0644)

		err := v.LoadSchemasFromDir(tmpDir)
		if err == nil {
			t.Error("expected error for invalid schema in directory")
		}
	})

	t.Run("LoadSchemasFromDir read failure", func(t *testing.T) {
		tmpDir, _ := os.MkdirTemp("", "unreadable-schemas")
		defer os.RemoveAll(tmpDir)
		path := filepath.Join(tmpDir, "unreadable.json")
		os.WriteFile(path, []byte("{}"), 0000)

		err := v.LoadSchemasFromDir(tmpDir)
		if err == nil {
			t.Error("expected error for unreadable schema file")
		}
	})
}
