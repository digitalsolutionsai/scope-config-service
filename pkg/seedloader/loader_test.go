package seedloader

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	configv1 "github.com/digitalsolutionsai/scope-config-service/proto/config/v1"
)

// mockApplier is a mock implementation of TemplateApplier for testing
type mockApplier struct {
	appliedTemplates []*configv1.ApplyConfigTemplateRequest
	shouldError      bool
}

func (m *mockApplier) ApplyConfigTemplate(ctx context.Context, req *configv1.ApplyConfigTemplateRequest) (*configv1.ConfigTemplate, error) {
	if m.shouldError {
		return nil, context.DeadlineExceeded
	}
	m.appliedTemplates = append(m.appliedTemplates, req)
	return req.Template, nil
}

func TestLoader_LoadAndApplyAll_NonExistentDirectory(t *testing.T) {
	applier := &mockApplier{}
	loader := NewLoader("/non/existent/directory", applier)

	err := loader.LoadAndApplyAll(context.Background())
	if err != nil {
		t.Errorf("Expected no error for non-existent directory, got: %v", err)
	}

	if len(applier.appliedTemplates) != 0 {
		t.Errorf("Expected no templates to be applied, got: %d", len(applier.appliedTemplates))
	}
}

func TestLoader_LoadAndApplyAll_EmptyDirectory(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "seedloader-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	applier := &mockApplier{}
	loader := NewLoader(tmpDir, applier)

	err = loader.LoadAndApplyAll(context.Background())
	if err != nil {
		t.Errorf("Expected no error for empty directory, got: %v", err)
	}

	if len(applier.appliedTemplates) != 0 {
		t.Errorf("Expected no templates to be applied, got: %d", len(applier.appliedTemplates))
	}
}

func TestLoader_LoadAndApplyAll_SingleTemplate(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "seedloader-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	templateContent := `service:
  id: "test-service"
  label: "Test Service"

groups:
  - id: "test-group"
    label: "Test Group"
    description: "Test group description"
    fields:
      - path: "test-path"
        label: "Test Label"
        description: "Test description"
        type: "STRING"
        defaultValue: "default"
        displayOn:
          - "PROJECT"
`

	templatePath := filepath.Join(tmpDir, "test.yaml")
	if err := os.WriteFile(templatePath, []byte(templateContent), 0644); err != nil {
		t.Fatalf("Failed to write template file: %v", err)
	}

	applier := &mockApplier{}
	loader := NewLoader(tmpDir, applier)

	err = loader.LoadAndApplyAll(context.Background())
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if len(applier.appliedTemplates) != 1 {
		t.Errorf("Expected 1 template to be applied, got: %d", len(applier.appliedTemplates))
	}

	if applier.appliedTemplates[0].Template.Identifier.ServiceName != "test-service" {
		t.Errorf("Expected service name 'test-service', got: %s", applier.appliedTemplates[0].Template.Identifier.ServiceName)
	}

	if applier.appliedTemplates[0].Template.Identifier.GroupId != "test-group" {
		t.Errorf("Expected group id 'test-group', got: %s", applier.appliedTemplates[0].Template.Identifier.GroupId)
	}

	if len(applier.appliedTemplates[0].Template.Fields) != 1 {
		t.Errorf("Expected 1 field, got: %d", len(applier.appliedTemplates[0].Template.Fields))
	}

	if applier.appliedTemplates[0].User != "system" {
		t.Errorf("Expected user 'system', got: %s", applier.appliedTemplates[0].User)
	}
}

func TestLoader_LoadAndApplyAll_RecursiveDirectory(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "seedloader-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	subDir := filepath.Join(tmpDir, "subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdir: %v", err)
	}

	templateContent1 := `service:
  id: "service-1"
  label: "Service 1"

groups:
  - id: "group-1"
    label: "Group 1"
    fields:
      - path: "field-1"
        label: "Field 1"
        type: "STRING"
        displayOn:
          - "PROJECT"
`

	templateContent2 := `service:
  id: "service-2"
  label: "Service 2"

groups:
  - id: "group-2"
    label: "Group 2"
    fields:
      - path: "field-2"
        label: "Field 2"
        type: "INT"
        displayOn:
          - "SYSTEM"
`

	if err := os.WriteFile(filepath.Join(tmpDir, "template1.yml"), []byte(templateContent1), 0644); err != nil {
		t.Fatalf("Failed to write template file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "template2.yaml"), []byte(templateContent2), 0644); err != nil {
		t.Fatalf("Failed to write template file: %v", err)
	}

	applier := &mockApplier{}
	loader := NewLoader(tmpDir, applier)

	err = loader.LoadAndApplyAll(context.Background())
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if len(applier.appliedTemplates) != 2 {
		t.Errorf("Expected 2 templates to be applied, got: %d", len(applier.appliedTemplates))
	}
}

func TestLoader_LoadAndApplyAll_MultipleGroups(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "seedloader-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	templateContent := `service:
  id: "multi-group-service"
  label: "Multi Group Service"

groups:
  - id: "group-1"
    label: "Group 1"
    fields:
      - path: "field-1"
        label: "Field 1"
        type: "STRING"
        displayOn:
          - "PROJECT"
  - id: "group-2"
    label: "Group 2"
    fields:
      - path: "field-2"
        label: "Field 2"
        type: "BOOLEAN"
        displayOn:
          - "SYSTEM"
`

	if err := os.WriteFile(filepath.Join(tmpDir, "multi.yaml"), []byte(templateContent), 0644); err != nil {
		t.Fatalf("Failed to write template file: %v", err)
	}

	applier := &mockApplier{}
	loader := NewLoader(tmpDir, applier)

	err = loader.LoadAndApplyAll(context.Background())
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if len(applier.appliedTemplates) != 2 {
		t.Errorf("Expected 2 templates to be applied (one per group), got: %d", len(applier.appliedTemplates))
	}
}

func TestLoader_LoadAndApplyAll_SecretFieldType(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "seedloader-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	templateContent := `service:
  id: "secret-test"
  label: "Secret Test Service"

groups:
  - id: "secrets"
    label: "Secret Fields"
    fields:
      - path: "api-key"
        label: "API Key"
        description: "Sensitive API key"
        type: "SECRET"
        defaultValue: ""
        displayOn:
          - "PROJECT"
`

	if err := os.WriteFile(filepath.Join(tmpDir, "secret.yaml"), []byte(templateContent), 0644); err != nil {
		t.Fatalf("Failed to write template file: %v", err)
	}

	applier := &mockApplier{}
	loader := NewLoader(tmpDir, applier)

	err = loader.LoadAndApplyAll(context.Background())
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if len(applier.appliedTemplates) != 1 {
		t.Errorf("Expected 1 template, got: %d", len(applier.appliedTemplates))
	}

	field := applier.appliedTemplates[0].Template.Fields[0]
	if field.Type != configv1.FieldType_SECRET {
		t.Errorf("Expected field type SECRET, got: %v", field.Type)
	}
}

func TestLoader_LoadAndApplyAll_InvalidYaml(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "seedloader-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	invalidContent := `service:
  id: "test"
  label: "Test
  invalid yaml content without closing quote
`

	if err := os.WriteFile(filepath.Join(tmpDir, "invalid.yaml"), []byte(invalidContent), 0644); err != nil {
		t.Fatalf("Failed to write template file: %v", err)
	}

	applier := &mockApplier{}
	loader := NewLoader(tmpDir, applier)

	err = loader.LoadAndApplyAll(context.Background())
	if err == nil {
		t.Error("Expected error for invalid YAML, got nil")
	}
}

func TestLoader_LoadAndApplyAll_MissingServiceId(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "seedloader-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	templateContent := `service:
  label: "Missing ID"

groups:
  - id: "group-1"
    label: "Group 1"
    fields:
      - path: "field-1"
        label: "Field 1"
        type: "STRING"
        displayOn:
          - "PROJECT"
`

	if err := os.WriteFile(filepath.Join(tmpDir, "missing-id.yaml"), []byte(templateContent), 0644); err != nil {
		t.Fatalf("Failed to write template file: %v", err)
	}

	applier := &mockApplier{}
	loader := NewLoader(tmpDir, applier)

	err = loader.LoadAndApplyAll(context.Background())
	if err == nil {
		t.Error("Expected error for missing service.id, got nil")
	}
}

func TestLoader_LoadAndApplyAll_FieldOptions(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "seedloader-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	templateContent := `service:
  id: "options-test"
  label: "Options Test"

groups:
  - id: "settings"
    label: "Settings"
    fields:
      - path: "log-level"
        label: "Log Level"
        type: "STRING"
        defaultValue: "INFO"
        options:
          - value: "DEBUG"
            label: "Debug"
          - value: "INFO"
            label: "Info"
          - value: "ERROR"
            label: "Error"
        displayOn:
          - "PROJECT"
`

	if err := os.WriteFile(filepath.Join(tmpDir, "options.yaml"), []byte(templateContent), 0644); err != nil {
		t.Fatalf("Failed to write template file: %v", err)
	}

	applier := &mockApplier{}
	loader := NewLoader(tmpDir, applier)

	err = loader.LoadAndApplyAll(context.Background())
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if len(applier.appliedTemplates) != 1 {
		t.Fatalf("Expected 1 template, got: %d", len(applier.appliedTemplates))
	}

	field := applier.appliedTemplates[0].Template.Fields[0]
	if len(field.Options) != 3 {
		t.Errorf("Expected 3 options, got: %d", len(field.Options))
	}

	if field.Options[0].Value != "DEBUG" || field.Options[0].Label != "Debug" {
		t.Errorf("First option mismatch, got: %v", field.Options[0])
	}
}

func TestLoader_LoadAndApplyAll_SortOrder(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "seedloader-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	templateContent := `service:
  id: "sort-order-test"
  label: "Sort Order Test"

groups:
  - id: "sorted-group"
    label: "Sorted Group"
    sortOrder: 100000
    fields:
      - path: "field-a"
        label: "Field A"
        type: "STRING"
        sortOrder: 200000
        displayOn:
          - "PROJECT"
      - path: "field-b"
        label: "Field B"
        type: "INT"
        sortOrder: 100000
        displayOn:
          - "PROJECT"
`

	if err := os.WriteFile(filepath.Join(tmpDir, "sortorder.yaml"), []byte(templateContent), 0644); err != nil {
		t.Fatalf("Failed to write template file: %v", err)
	}

	applier := &mockApplier{}
	loader := NewLoader(tmpDir, applier)

	err = loader.LoadAndApplyAll(context.Background())
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if len(applier.appliedTemplates) != 1 {
		t.Fatalf("Expected 1 template, got: %d", len(applier.appliedTemplates))
	}

	template := applier.appliedTemplates[0].Template
	if template.SortOrder != 100000 {
		t.Errorf("Expected group sort order 100000, got: %d", template.SortOrder)
	}

	if len(template.Fields) != 2 {
		t.Fatalf("Expected 2 fields, got: %d", len(template.Fields))
	}

	// Check first field sort order
	if template.Fields[0].SortOrder != 200000 {
		t.Errorf("Expected first field sort order 200000, got: %d", template.Fields[0].SortOrder)
	}

	// Check second field sort order
	if template.Fields[1].SortOrder != 100000 {
		t.Errorf("Expected second field sort order 100000, got: %d", template.Fields[1].SortOrder)
	}
}

func TestToFieldType(t *testing.T) {
	tests := []struct {
		input    string
		expected configv1.FieldType
	}{
		{"STRING", configv1.FieldType_STRING},
		{"INT", configv1.FieldType_INT},
		{"FLOAT", configv1.FieldType_FLOAT},
		{"BOOLEAN", configv1.FieldType_BOOLEAN},
		{"JSON", configv1.FieldType_JSON},
		{"ARRAY_STRING", configv1.FieldType_ARRAY_STRING},
		{"SECRET", configv1.FieldType_SECRET},
		{"INVALID", configv1.FieldType_FIELD_TYPE_UNSPECIFIED},
		{"", configv1.FieldType_FIELD_TYPE_UNSPECIFIED},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := toFieldType(tt.input)
			if result != tt.expected {
				t.Errorf("toFieldType(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestToScope(t *testing.T) {
	tests := []struct {
		input    string
		expected configv1.Scope
	}{
		{"SYSTEM", configv1.Scope_SYSTEM},
		{"PROJECT", configv1.Scope_PROJECT},
		{"STORE", configv1.Scope_STORE},
		{"USER", configv1.Scope_USER},
		{"INVALID", configv1.Scope_SCOPE_UNSPECIFIED},
		{"", configv1.Scope_SCOPE_UNSPECIFIED},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := toScope(tt.input)
			if result != tt.expected {
				t.Errorf("toScope(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestLoader_LoadAndApplyAll_IgnoresNonYamlFiles(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "seedloader-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create valid YAML template
	templateContent := `service:
  id: "test-service"
  label: "Test Service"

groups:
  - id: "test-group"
    label: "Test Group"
    fields:
      - path: "test-path"
        label: "Test Label"
        type: "STRING"
        displayOn:
          - "PROJECT"
`

	if err := os.WriteFile(filepath.Join(tmpDir, "valid.yaml"), []byte(templateContent), 0644); err != nil {
		t.Fatalf("Failed to write template file: %v", err)
	}

	// Create non-YAML files that should be ignored
	if err := os.WriteFile(filepath.Join(tmpDir, "readme.txt"), []byte("This is a readme"), 0644); err != nil {
		t.Fatalf("Failed to write txt file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "config.json"), []byte(`{"key": "value"}`), 0644); err != nil {
		t.Fatalf("Failed to write json file: %v", err)
	}

	applier := &mockApplier{}
	loader := NewLoader(tmpDir, applier)

	err = loader.LoadAndApplyAll(context.Background())
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if len(applier.appliedTemplates) != 1 {
		t.Errorf("Expected 1 template to be applied (ignoring non-YAML files), got: %d", len(applier.appliedTemplates))
	}
}
