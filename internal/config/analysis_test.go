package config

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestAnalysisConfig(t *testing.T) {
	for _, k := range []string{"APP_ENV", "DATABASE_URL", "AUTH_USERNAME", "JWT_SECRET", "PUBLIC_ORIGIN", "LISTEN_ADDR", "ANTHROPIC_API_KEY", "LEETGRINDER_ANALYSIS_MODEL", "LEETGRINDER_ANALYSIS_DAILY_LIMIT"} {
		t.Setenv(k, "")
	}
	c, err := Load()
	if err != nil || c.AnthropicAPIKey.Reveal() != "" || c.AnalysisModel != "claude-sonnet-5" || c.AnalysisDailyLimit != 50 {
		t.Fatalf("defaults: %+v %v", c, err)
	}
	const key = "sk-ant-test-never-print"
	t.Setenv("ANTHROPIC_API_KEY", " "+key+"\n")
	t.Setenv("LEETGRINDER_ANALYSIS_MODEL", "claude-opus-5")
	t.Setenv("LEETGRINDER_ANALYSIS_DAILY_LIMIT", "0")
	if c, err = Load(); err != nil || c.AnthropicAPIKey.Reveal() != key || c.AnalysisModel != "claude-opus-5" || c.AnalysisDailyLimit != 0 {
		t.Fatalf("configured: %+v %v", c, err)
	}
	encoded, _ := json.Marshal(c)
	for _, printed := range []string{fmt.Sprint(c), fmt.Sprintf("%+v", c), fmt.Sprintf("%#v", c), fmt.Sprint(c.AnthropicAPIKey), string(encoded)} {
		if strings.Contains(printed, key) {
			t.Fatalf("key printed: %s", printed)
		}
	}
	for name, value := range map[string]string{"LEETGRINDER_ANALYSIS_DAILY_LIMIT": "-1", "LEETGRINDER_ANALYSIS_MODEL": "claude sonnet"} {
		t.Setenv(name, value)
		if _, err = Load(); err == nil {
			t.Errorf("%s=%q accepted", name, value)
		}
		t.Setenv(name, "")
	}
	t.Setenv("ANTHROPIC_API_KEY", "two words")
	if _, err = Load(); err == nil || strings.Contains(err.Error(), "two words") {
		t.Fatalf("bad key: %v", err)
	}
}
