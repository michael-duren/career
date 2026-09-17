package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	DatabaseURL, Username, PasswordHash, JWTSecret, PublicOrigin, ListenAddr, StaticDir string
	Production                                                                          bool
	// OTelEndpoint is the OTLP gRPC endpoint; empty means export metrics to stdout.
	OTelEndpoint, OTelServiceName string
	OTelExportInterval            time.Duration
}

func Load() (Config, error) {
	c := Config{
		DatabaseURL:        os.Getenv("DATABASE_URL"),
		Username:           os.Getenv("AUTH_USERNAME"),
		PasswordHash:       os.Getenv("AUTH_PASSWORD_HASH"),
		JWTSecret:          os.Getenv("JWT_SECRET"),
		PublicOrigin:       os.Getenv("PUBLIC_ORIGIN"),
		ListenAddr:         os.Getenv("LISTEN_ADDR"),
		StaticDir:          os.Getenv("STATIC_DIR"),
		Production:         os.Getenv("APP_ENV") == "production",
		OTelServiceName:    os.Getenv("OTEL_SERVICE_NAME"),
		OTelExportInterval: 10 * time.Second,
	}

	// Signal-specific endpoint wins, matching the OTel SDK. An endpoint set to
	// empty is honored so local runs can opt out of the collector.
	endpoint, endpointSet := os.LookupEnv("OTEL_EXPORTER_OTLP_METRICS_ENDPOINT")
	if !endpointSet {
		endpoint, endpointSet = os.LookupEnv("OTEL_EXPORTER_OTLP_ENDPOINT")
	}
	c.OTelEndpoint = endpoint
	if c.OTelServiceName == "" {
		c.OTelServiceName = "career-strategy"
	}
	if v := os.Getenv("OTEL_METRIC_EXPORT_INTERVAL"); v != "" {
		ms, err := strconv.Atoi(v)
		if err != nil || ms <= 0 {
			return c, fmt.Errorf("OTEL_METRIC_EXPORT_INTERVAL must be a positive number of milliseconds")
		}
		c.OTelExportInterval = time.Duration(ms) * time.Millisecond
	}
	if !c.Production {
		if !endpointSet {
			c.OTelEndpoint = "http://localhost:4317"
		}
		if c.DatabaseURL == "" {
			c.DatabaseURL = "postgres://career_dev:career_dev_local@127.0.0.1:5433/career_dev?sslmode=disable"
		}
		if c.ListenAddr == "" {
			c.ListenAddr = "127.0.0.1:8080"
		}
		if c.PublicOrigin == "" {
			c.PublicOrigin = "http://localhost:8080"
		}
		if c.Username == "" {
			c.Username = "admin"
		}
		if c.JWTSecret == "" {
			c.JWTSecret = "local-development-only-session-secret"
		}
	}
	if c.StaticDir == "" {
		c.StaticDir = "dist"
	}
	for name, v := range map[string]string{"DATABASE_URL": c.DatabaseURL, "LISTEN_ADDR": c.ListenAddr, "PUBLIC_ORIGIN": c.PublicOrigin, "AUTH_USERNAME": c.Username, "JWT_SECRET": c.JWTSecret} {
		if v == "" {
			return c, fmt.Errorf("%s is required", name)
		}
	}
	u, err := url.Parse(c.PublicOrigin)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
		return c, fmt.Errorf("PUBLIC_ORIGIN must be an absolute origin")
	}
	c.PublicOrigin = strings.TrimSuffix(c.PublicOrigin, "/")
	if c.Production && (len(c.JWTSecret) < 32 || strings.Contains(c.JWTSecret, "local-development") || strings.Contains(c.JWTSecret, "your-secure") || c.PasswordHash == "" || strings.Contains(c.PasswordHash, "YourHashed")) {
		return c, fmt.Errorf("production requires a bcrypt AUTH_PASSWORD_HASH and a strong JWT_SECRET (32+ characters)")
	}
	return c, nil
}
