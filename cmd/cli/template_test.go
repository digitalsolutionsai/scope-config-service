package main

import (
	"testing"

	configv1 "github.com/digitalsolutionsai/scope-config-service/proto/config/v1"
)

func TestToFieldType_Valid(t *testing.T) {
	tests := []struct {
		input    string
		expected configv1.FieldType
	}{
		{"STRING", configv1.FieldType_STRING},
		{"INT", configv1.FieldType_INT},
		{"BOOLEAN", configv1.FieldType_BOOLEAN},
		{"FLOAT", configv1.FieldType_FLOAT},
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

func TestToFieldType_Invalid(t *testing.T) {
	result := toFieldType("UNKNOWN")
	if result != configv1.FieldType_FIELD_TYPE_UNSPECIFIED {
		t.Errorf("toFieldType(UNKNOWN) = %v, want FIELD_TYPE_UNSPECIFIED", result)
	}
}

func TestToScope_Valid(t *testing.T) {
	tests := []struct {
		input    string
		expected configv1.Scope
	}{
		{"SYSTEM", configv1.Scope_SYSTEM},
		{"PROJECT", configv1.Scope_PROJECT},
		{"STORE", configv1.Scope_STORE},
		{"USER", configv1.Scope_USER},
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

func TestToScope_Invalid(t *testing.T) {
	result := toScope("INVALID")
	if result != configv1.Scope_SCOPE_UNSPECIFIED {
		t.Errorf("toScope(INVALID) = %v, want SCOPE_UNSPECIFIED", result)
	}
}
