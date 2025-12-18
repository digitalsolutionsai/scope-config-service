package scopeconfig

import (
	"testing"
	"time"
)

func TestWithCache(t *testing.T) {
	cfg := &clientConfig{}
	opt := WithCache(2 * time.Minute)
	opt(cfg)

	if !cfg.cacheEnabled {
		t.Error("Expected cacheEnabled to be true")
	}
	if cfg.cacheTTL != 2*time.Minute {
		t.Errorf("Expected cacheTTL to be 2 minutes, got %v", cfg.cacheTTL)
	}
}

func TestWithBackgroundSync(t *testing.T) {
	cfg := &clientConfig{}
	opt := WithBackgroundSync(45 * time.Second)
	opt(cfg)

	if !cfg.syncEnabled {
		t.Error("Expected syncEnabled to be true")
	}
	if cfg.syncInterval != 45*time.Second {
		t.Errorf("Expected syncInterval to be 45 seconds, got %v", cfg.syncInterval)
	}
}

func TestWithAddress(t *testing.T) {
	cfg := &clientConfig{}
	opt := WithAddress("localhost:50051")
	opt(cfg)

	if cfg.address != "localhost:50051" {
		t.Errorf("Expected address to be localhost:50051, got %s", cfg.address)
	}
}

func TestWithInsecure(t *testing.T) {
	cfg := &clientConfig{}
	opt := WithInsecure()
	opt(cfg)

	if len(cfg.dialOptions) != 1 {
		t.Error("Expected one dial option to be added")
	}
}
