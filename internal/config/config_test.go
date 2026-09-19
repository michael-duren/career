package config

import (
	"os"
	"testing"
	"time"
)

func TestLocalDefaultsAndProductionRequired(t *testing.T) {
	for _, k := range []string{"APP_ENV", "DATABASE_URL", "AUTH_USERNAME", "AUTH_PASSWORD_HASH", "JWT_SECRET", "PUBLIC_ORIGIN", "LISTEN_ADDR", "STATIC_DIR"} {
		t.Setenv(k, "")
	}
	c, err := Load()
	if err != nil || c.ListenAddr != "127.0.0.1:8080" || c.DatabaseURL == "" {
		t.Fatalf("%+v %v", c, err)
	}
	t.Setenv("APP_ENV", "production")
	if _, err = Load(); err == nil {
		t.Fatal("production accepted missing settings")
	}
	t.Setenv("APP_ENV", "")
	t.Setenv("PUBLIC_ORIGIN", "https://example.com/path")
	if _, err = Load(); err == nil {
		t.Fatal("accepted path in origin")
	}
}

func TestOTelFallbacks(t *testing.T) {
	for _, k := range []string{"APP_ENV", "DATABASE_URL", "AUTH_USERNAME", "JWT_SECRET", "PUBLIC_ORIGIN", "LISTEN_ADDR", "OTEL_SERVICE_NAME", "OTEL_METRIC_EXPORT_INTERVAL"} {
		t.Setenv(k, "")
	}
	for _, k := range []string{"OTEL_EXPORTER_OTLP_ENDPOINT", "OTEL_EXPORTER_OTLP_METRICS_ENDPOINT"} {
		t.Setenv(k, "")
		os.Unsetenv(k)
	}
	c, err := Load()
	if err != nil || c.OTelEndpoint != "http://localhost:4317" || c.OTelServiceName != "career-strategy" || c.OTelExportInterval != 10*time.Second {
		t.Fatalf("local defaults: %+v %v", c, err)
	}
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")
	if c, _ = Load(); c.OTelEndpoint != "" {
		t.Fatalf("explicit empty endpoint not honored: %q", c.OTelEndpoint)
	}
	t.Setenv("OTEL_EXPORTER_OTLP_METRICS_ENDPOINT", "https://collector.example:4317")
	t.Setenv("OTEL_METRIC_EXPORT_INTERVAL", "2500")
	if c, _ = Load(); c.OTelEndpoint != "https://collector.example:4317" || c.OTelExportInterval != 2500*time.Millisecond {
		t.Fatalf("env overrides: %+v", c)
	}
	t.Setenv("OTEL_METRIC_EXPORT_INTERVAL", "soon")
	if _, err = Load(); err == nil {
		t.Fatal("accepted invalid export interval")
	}
}
