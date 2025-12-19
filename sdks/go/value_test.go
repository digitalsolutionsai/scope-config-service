package scopeconfig

import (
	"testing"
)

func TestGetValueOptions(t *testing.T) {
	opts := &GetValueOptions{
		UseDefault: true,
		Inherit:    true,
	}

	if !opts.UseDefault {
		t.Error("Expected UseDefault to be true")
	}
	if !opts.Inherit {
		t.Error("Expected Inherit to be true")
	}
}

func TestGetValueOptionsDefault(t *testing.T) {
	opts := &GetValueOptions{}

	if opts.UseDefault {
		t.Error("Expected UseDefault to be false by default")
	}
	if opts.Inherit {
		t.Error("Expected Inherit to be false by default")
	}
}

// Note: Full GetValue tests require a running server or mocks
// The integration tests in tests/integration_test.go cover the full flow
