package config

import (
	"bytes"
	"encoding/base64"
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

func TestLeetgrinderSecretKey(t *testing.T) {
	for _, k := range []string{"APP_ENV", "DATABASE_URL", "AUTH_USERNAME", "AUTH_PASSWORD_HASH", "PUBLIC_ORIGIN", "LISTEN_ADDR", "LEETGRINDER_SECRET_KEY"} {
		t.Setenv(k, "")
	}
	t.Setenv("JWT_SECRET", "first-development-secret")
	dev, err := Load()
	if err != nil || len(dev.LeetgrinderSecretKey) != 32 {
		t.Fatalf("dev key: %v %v", dev.LeetgrinderSecretKey, err)
	}
	again, _ := Load()
	t.Setenv("JWT_SECRET", "second-development-secret")
	other, _ := Load()
	if !bytes.Equal(dev.LeetgrinderSecretKey, again.LeetgrinderSecretKey) || bytes.Equal(dev.LeetgrinderSecretKey, other.LeetgrinderSecretKey) {
		t.Fatal("dev key is not derived from JWT_SECRET")
	}
	key := bytes.Repeat([]byte{9}, 32)
	t.Setenv("LEETGRINDER_SECRET_KEY", base64.StdEncoding.EncodeToString(key))
	if c, err := Load(); err != nil || !bytes.Equal(c.LeetgrinderSecretKey, key) {
		t.Fatalf("explicit key: %v", err)
	}
	for _, bad := range []string{"not base64!", base64.StdEncoding.EncodeToString(key[:16])} {
		t.Setenv("LEETGRINDER_SECRET_KEY", bad)
		if _, err := Load(); err == nil {
			t.Fatalf("accepted key %q", bad)
		}
	}
	t.Setenv("LEETGRINDER_SECRET_KEY", "")
	t.Setenv("APP_ENV", "production")
	t.Setenv("PUBLIC_ORIGIN", "https://example.com")
	t.Setenv("JWT_SECRET", "a-production-secret-that-is-long-enough-1234")
	t.Setenv("AUTH_PASSWORD_HASH", "$2b$10$DgkrAm6JRG906Gr0BZZ9C.wEBc9Yg0Wm.QetZAgR.20UsQ12ekaHK")
	t.Setenv("DATABASE_URL", "postgres://db/career")
	t.Setenv("LISTEN_ADDR", ":8080")
	t.Setenv("AUTH_USERNAME", "admin")
	prod, err := Load()
	if err != nil || prod.LeetgrinderSecretKey != nil {
		t.Fatalf("production without a key must load with no key: %v", err)
	}
	if prod.RequireLeetgrinderSecret(false) != nil {
		t.Fatal("no stored token should not need a key")
	}
	if prod.RequireLeetgrinderSecret(true) == nil {
		t.Fatal("stored token without a key was accepted")
	}
	if dev.RequireLeetgrinderSecret(true) != nil {
		t.Fatal("configured key rejected")
	}
}
