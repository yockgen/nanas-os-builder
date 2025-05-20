package validate

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

// loadFile reads a test JSON file from the project root testdata directory.
func loadFile(t *testing.T, relPath string) []byte {
	t.Helper()
	// Determine project root relative to this test file
	root := filepath.Join("..", "..")
	fullPath := filepath.Join(root, relPath)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		t.Fatalf("failed to read file %s: %v", fullPath, err)
	}
	return data
}

func TestValid(t *testing.T) {
	v := loadFile(t, "testdata/valid.json")
	if err := ValidateComposerJSON(v); err != nil {
		t.Errorf("expected valid.json to pass, but got: %v", err)
	}
}

func TestInvalid(t *testing.T) {
	v := loadFile(t, "testdata/invalid.json")
	if err := ValidateComposerJSON(v); err == nil {
		t.Errorf("expected invalid.json to fail validation")
	}
}

func TestValidImage(t *testing.T) {
	v := loadFile(t, "image-templates/default-image-template.yml")

	// Parse to generic JSON interface
	var raw interface{}
	if err := yaml.Unmarshal(v, &raw); err != nil {
		t.Errorf("yml parsing error: %v", err)
	}

	// 3) Re‐marshal to JSON bytes
	dataJSON, err := json.Marshal(raw)
	if err != nil {
		t.Errorf("json marshaling error: %v", err)
	}
	if err := ValidateImageJSON(dataJSON); err != nil {
		t.Errorf("expected image-templates/default-image-template.yml to pass, but got: %v", err)
	}
}

func TestInvalidImage(t *testing.T) {
	v := loadFile(t, "testdata/invalid-image.yml")

	// Parse to generic JSON interface
	var raw interface{}
	if err := yaml.Unmarshal(v, &raw); err != nil {
		t.Errorf("yml parsing error: %v", err)
	}

	// 3) Re‐marshal to JSON bytes
	dataJSON, err := json.Marshal(raw)
	if err != nil {
		t.Errorf("json marshaling error: %v", err)
	}
	if err := ValidateImageJSON(dataJSON); err == nil {
		t.Errorf("expected testdata/invalid-image.yml to pass, but got: %v", err)
	}
}
