package scopeconfig

import (
	"testing"
)

func TestParseAndValidateTemplate(t *testing.T) {
	tests := []struct {
		name        string
		yaml        string
		expectError bool
		serviceName string
		groupCount  int
	}{
		{
			name: "valid template with one group",
			yaml: `
service:
  id: "test-service"
  label: "Test Service"
groups:
  - id: "database"
    label: "Database Config"
    fields:
      - path: "host"
        label: "Host"
        type: "STRING"
        defaultValue: "localhost"
`,
			expectError: false,
			serviceName: "test-service",
			groupCount:  1,
		},
		{
			name: "valid template with multiple groups",
			yaml: `
service:
  id: "multi-service"
  label: "Multi Service"
groups:
  - id: "database"
    label: "Database"
    fields: []
  - id: "logging"
    label: "Logging"
    fields: []
`,
			expectError: false,
			serviceName: "multi-service",
			groupCount:  2,
		},
		{
			name: "missing service id",
			yaml: `
service:
  label: "Test Service"
groups:
  - id: "database"
    label: "Database"
    fields: []
`,
			expectError: true,
		},
		{
			name:        "invalid yaml",
			yaml:        `{invalid: yaml: content`,
			expectError: true,
		},
		{
			name: "template with field options",
			yaml: `
service:
  id: "options-service"
  label: "Options Service"
groups:
  - id: "settings"
    label: "Settings"
    fields:
      - path: "log_level"
        label: "Log Level"
        type: "STRING"
        defaultValue: "INFO"
        options:
          - value: "DEBUG"
            label: "Debug"
          - value: "INFO"
            label: "Info"
`,
			expectError: false,
			serviceName: "options-service",
			groupCount:  1,
		},
		{
			name: "template with displayOn scopes",
			yaml: `
service:
  id: "scoped-service"
  label: "Scoped Service"
groups:
  - id: "config"
    label: "Config"
    fields:
      - path: "setting"
        label: "Setting"
        type: "STRING"
        displayOn:
          - "SYSTEM"
          - "PROJECT"
`,
			expectError: false,
			serviceName: "scoped-service",
			groupCount:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			template, err := parseAndValidateTemplate([]byte(tt.yaml))

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if template.Service.Name != tt.serviceName {
				t.Errorf("Expected service name '%s', got '%s'", tt.serviceName, template.Service.Name)
			}

			if len(template.Groups) != tt.groupCount {
				t.Errorf("Expected %d groups, got %d", tt.groupCount, len(template.Groups))
			}
		})
	}
}

func TestToFieldType(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"STRING", "STRING"},
		{"INT", "INT"},
		{"FLOAT", "FLOAT"},
		{"BOOLEAN", "BOOLEAN"},
		{"JSON", "JSON"},
		{"ARRAY_STRING", "ARRAY_STRING"},
		{"SECRET", "SECRET"},
		{"INVALID", "FIELD_TYPE_UNSPECIFIED"},
		{"", "FIELD_TYPE_UNSPECIFIED"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := toFieldType(tt.input)
			if result.String() != tt.expected {
				t.Errorf("toFieldType(%s) = %v, expected %s", tt.input, result.String(), tt.expected)
			}
		})
	}
}

func TestToScope(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"SYSTEM", "SYSTEM"},
		{"PROJECT", "PROJECT"},
		{"STORE", "STORE"},
		{"USER", "USER"},
		{"INVALID", "SCOPE_UNSPECIFIED"},
		{"", "SCOPE_UNSPECIFIED"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := toScope(tt.input)
			if result.String() != tt.expected {
				t.Errorf("toScope(%s) = %v, expected %s", tt.input, result.String(), tt.expected)
			}
		})
	}
}

func TestYamlTemplateStructure(t *testing.T) {
	yaml := `
service:
  id: "structured-service"
  label: "Structured Service"
groups:
  - id: "api"
    label: "API Configuration"
    description: "API settings"
    sortOrder: 100000
    fields:
      - path: "timeout"
        label: "Request Timeout"
        description: "HTTP request timeout in seconds"
        type: "INT"
        defaultValue: "30"
        sortOrder: 100000
        displayOn:
          - "SYSTEM"
          - "PROJECT"
`

	template, err := parseAndValidateTemplate([]byte(yaml))
	if err != nil {
		t.Fatalf("Failed to parse template: %v", err)
	}

	// Verify service
	if template.Service.Name != "structured-service" {
		t.Errorf("Expected service name 'structured-service', got '%s'", template.Service.Name)
	}
	if template.Service.Label != "Structured Service" {
		t.Errorf("Expected service label 'Structured Service', got '%s'", template.Service.Label)
	}

	// Verify group
	if len(template.Groups) != 1 {
		t.Fatalf("Expected 1 group, got %d", len(template.Groups))
	}

	group := template.Groups[0]
	if group.ID != "api" {
		t.Errorf("Expected group ID 'api', got '%s'", group.ID)
	}
	if group.Label != "API Configuration" {
		t.Errorf("Expected group label 'API Configuration', got '%s'", group.Label)
	}
	if group.Description != "API settings" {
		t.Errorf("Expected group description 'API settings', got '%s'", group.Description)
	}
	if group.SortOrder != 100000 {
		t.Errorf("Expected group sortOrder 100000, got %d", group.SortOrder)
	}

	// Verify field
	if len(group.Fields) != 1 {
		t.Fatalf("Expected 1 field, got %d", len(group.Fields))
	}

	field := group.Fields[0]
	if field.Path != "timeout" {
		t.Errorf("Expected field path 'timeout', got '%s'", field.Path)
	}
	if field.Type != "INT" {
		t.Errorf("Expected field type 'INT', got '%s'", field.Type)
	}
	if field.DefaultValue != "30" {
		t.Errorf("Expected default value '30', got '%s'", field.DefaultValue)
	}
	if len(field.DisplayOn) != 2 {
		t.Errorf("Expected 2 displayOn scopes, got %d", len(field.DisplayOn))
	}
}
