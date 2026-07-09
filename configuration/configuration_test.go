package configuration_test

import (
	"context"
	"testing"

	"github.com/mtrace-project/mtrace/configuration"
	"github.com/mtrace-project/mtrace/configuration/jaeger"
	"github.com/mtrace-project/mtrace/configuration/openobserve"
)

func TestNewTraceAdapterFromConfig_OpenObserve(t *testing.T) {
	cfg := &configuration.AppConfig{
		BackendType: "openobserve",
		Verbose:     true,
		OpenObserveConfig: &openobserve.OpenObserveConfig{
			BaseURL:    "http://localhost:5080",
			OrgName:    "default",
			StreamName: "default",
			Username:   "admin@example.com",
			Password:   "admin",
		},
	}

	adapter, err := cfg.NewTraceAdapterFromConfig(context.Background())
	if err != nil {
		t.Fatalf("Unexpected error creating openobserve adapter: %v", err)
	}

	if adapter == nil {
		t.Fatal("Expected non-nil OpenObserveTraceAdapter")
	}
}

func TestNewTraceAdapterFromConfig_UnsupportedBackend(t *testing.T) {
	cfg := &configuration.AppConfig{
		BackendType: "unsupported_backend",
		Verbose:     true,
	}

	adapter, err := cfg.NewTraceAdapterFromConfig(context.Background())
	if err == nil {
		t.Fatal("Expected error for unsupported backend type, got nil")
	}

	if adapter != nil {
		t.Fatal("Expected nil adapter for unsupported backend type")
	}
}

func TestNewTraceAdapterFromConfig_Jaeger(t *testing.T) {
	cfg := &configuration.AppConfig{
		BackendType: "jaeger",
		Verbose:     true,
		JaegerConfig: &jaeger.JaegerConfig{
			BaseURL: "http://localhost:16686",
		},
	}

	adapter, err := cfg.NewTraceAdapterFromConfig(context.Background())
	if err != nil {
		t.Fatalf("Unexpected error creating jaeger adapter: %v", err)
	}

	if adapter == nil {
		t.Fatal("Expected non-nil JaegerTraceAdapter")
	}
}
