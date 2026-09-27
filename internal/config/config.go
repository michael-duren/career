package config

import (
	"crypto/hkdf"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	WhisperURL, WhisperModel                                                            string
	RunningTranscribeEnabled                                                            bool
	DatabaseURL, Username, PasswordHash, JWTSecret, PublicOrigin, ListenAddr, StaticDir string
	Production                                                                          bool
	// OTelEndpoint is the OTLP gRPC endpoint; empty means export metrics to stdout.
	OTelEndpoint, OTelServiceName string
	OTelExportInterval            time.Duration
	// LeetgrinderSecretKey encrypts the ntfy token. It is nil in production
	// when LEETGRINDER_SECRET_KEY is unset; serve refuses to start if a token
	// is already stored in that case.
	LeetgrinderSecretKey []byte
	// AnthropicAPIKey enables Leetgrinder complexity analysis. It is read
	// from ANTHROPIC_API_KEY only, never stored, and prints as [redacted].
	AnthropicAPIKey Secret
	// AnalysisModel is the model for complexity analysis.
	AnalysisModel string
	// AnalysisDailyLimit caps analysis requests per local day; 0 sends none.
	AnalysisDailyLimit int
}

// Secret is a credential that never prints its value.
type Secret string

func (Secret) String() string   { return "[redacted]" }
func (Secret) GoString() string { return "[redacted]" }

// MarshalText keeps the value out of JSON and other text encodings.
func (Secret) MarshalText() ([]byte, error) { return []byte("[redacted]"), nil }

// Reveal returns the credential for the one place that sends it.
func (s Secret) Reveal() string { return string(s) }

var analysisModel = regexp.MustCompile(`^[a-z0-9][a-z0-9._:@-]{0,99}$`)

func Load() (Config, error) {
	c := Config{
		WhisperURL: os.Getenv("WHISPER_URL"), WhisperModel: os.Getenv("WHISPER_MODEL"), RunningTranscribeEnabled: os.Getenv("RUNNING_TRANSCRIBE_ENABLED") == "true",
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

	if c.WhisperURL == "" {
		c.WhisperURL = "http://whisper:8080"
	}
	if c.WhisperModel == "" {
		c.WhisperModel = "ggml-base.en"
	}
	if u, err := url.Parse(c.WhisperURL); err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil {
		return c, fmt.Errorf("WHISPER_URL must be a local HTTP service URL")
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
	if err = c.loadAnalysis(); err != nil {
		return c, err
	}
	if c.LeetgrinderSecretKey, err = leetgrinderSecretKey(os.Getenv("LEETGRINDER_SECRET_KEY"), c.JWTSecret, c.Production); err != nil {
		return c, err
	}
	if c.Production && (len(c.JWTSecret) < 32 || strings.Contains(c.JWTSecret, "local-development") || strings.Contains(c.JWTSecret, "your-secure") || c.PasswordHash == "" || strings.Contains(c.PasswordHash, "YourHashed")) {
		return c, fmt.Errorf("production requires a bcrypt AUTH_PASSWORD_HASH and a strong JWT_SECRET (32+ characters)")
	}
	return c, nil
}

// loadAnalysis reads the Leetgrinder complexity analysis settings. Its errors
// never contain the API key.
func (c *Config) loadAnalysis() error {
	key := strings.TrimSpace(os.Getenv("ANTHROPIC_API_KEY"))
	if strings.IndexFunc(key, func(r rune) bool { return r <= ' ' || r == 0x7f }) >= 0 || len(key) > 512 {
		return fmt.Errorf("ANTHROPIC_API_KEY must be a single token without spaces")
	}
	c.AnthropicAPIKey = Secret(key)
	c.AnalysisModel = strings.TrimSpace(os.Getenv("LEETGRINDER_ANALYSIS_MODEL"))
	if c.AnalysisModel == "" {
		c.AnalysisModel = "claude-sonnet-5"
	}
	if !analysisModel.MatchString(c.AnalysisModel) {
		return fmt.Errorf("LEETGRINDER_ANALYSIS_MODEL must be a model ID such as claude-sonnet-5")
	}
	c.AnalysisDailyLimit = 50
	if v := strings.TrimSpace(os.Getenv("LEETGRINDER_ANALYSIS_DAILY_LIMIT")); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 || n > 10000 {
			return fmt.Errorf("LEETGRINDER_ANALYSIS_DAILY_LIMIT must be a whole number from 0 to 10000")
		}
		c.AnalysisDailyLimit = n
	}
	return nil
}

// leetgrinderSecretKey decodes a base64 32-byte key. Development derives one
// from JWT_SECRET so local runs need no extra setup.
func leetgrinderSecretKey(encoded, jwtSecret string, production bool) ([]byte, error) {
	if encoded == "" {
		if production {
			return nil, nil
		}
		return hkdf.Key(sha256.New, []byte(jwtSecret), nil, "career leetgrinder secret key", 32)
	}
	for _, enc := range []*base64.Encoding{base64.StdEncoding, base64.RawStdEncoding, base64.URLEncoding, base64.RawURLEncoding} {
		if key, err := enc.DecodeString(encoded); err == nil && len(key) == 32 {
			return key, nil
		}
	}
	return nil, fmt.Errorf("LEETGRINDER_SECRET_KEY must be 32 bytes encoded as base64")
}

// RequireLeetgrinderSecret fails when there is no key but an encrypted ntfy
// token is already stored, since that token could never be decrypted.
func (c Config) RequireLeetgrinderSecret(tokenStored bool) error {
	if len(c.LeetgrinderSecretKey) == 0 && tokenStored {
		return fmt.Errorf("LEETGRINDER_SECRET_KEY is required because an encrypted ntfy token is stored")
	}
	return nil
}
