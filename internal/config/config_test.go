package config

import "testing"

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
