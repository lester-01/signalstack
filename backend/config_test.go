package main

import (
	"reflect"
	"testing"
)

func env(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func TestLoadConfigRequiresAuthToken(t *testing.T) {
	if _, err := LoadConfig(env(nil)); err == nil {
		t.Fatal("expected an error when AUTH_TOKEN is unset")
	}
}

func TestLoadConfigDefaults(t *testing.T) {
	cfg, err := LoadConfig(env(map[string]string{"AUTH_TOKEN": "x"}))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Addr != ":8080" {
		t.Errorf("Addr = %q, want :8080", cfg.Addr)
	}
	if !reflect.DeepEqual(cfg.AllowedOrigins, []string{"http://localhost:3000"}) {
		t.Errorf("AllowedOrigins = %v", cfg.AllowedOrigins)
	}
	if cfg.RateLimit != 10 || cfg.MetricsEnabled {
		t.Errorf("RateLimit = %v, MetricsEnabled = %v; want 10, false", cfg.RateLimit, cfg.MetricsEnabled)
	}
	if len(cfg.SessionSecret) != 32 || !cfg.SessionSecretGenerated {
		t.Errorf("expected a generated 32-byte session secret, got %d bytes (generated=%v)", len(cfg.SessionSecret), cfg.SessionSecretGenerated)
	}
}

func TestLoadConfigOverrides(t *testing.T) {
	cfg, err := LoadConfig(env(map[string]string{
		"AUTH_TOKEN":      "x",
		"PORT":            "9000",
		"ALLOWED_ORIGINS": " https://a.example , https://b.example ,",
		"SESSION_SECRET":  "s3cret",
		"RATE_LIMIT":      "0",
		"METRICS_ENABLED": "true",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Addr != ":9000" {
		t.Errorf("Addr = %q, want :9000", cfg.Addr)
	}
	if !reflect.DeepEqual(cfg.AllowedOrigins, []string{"https://a.example", "https://b.example"}) {
		t.Errorf("AllowedOrigins = %v", cfg.AllowedOrigins)
	}
	if string(cfg.SessionSecret) != "s3cret" || cfg.SessionSecretGenerated {
		t.Errorf("SESSION_SECRET not honoured")
	}
	if cfg.RateLimit != 0 || !cfg.MetricsEnabled {
		t.Errorf("RATE_LIMIT/METRICS_ENABLED not honoured: %v, %v", cfg.RateLimit, cfg.MetricsEnabled)
	}

	cfg, _ = LoadConfig(env(map[string]string{"AUTH_TOKEN": "x", "ADDR": "127.0.0.1:1", "PORT": "9000"}))
	if cfg.Addr != "127.0.0.1:1" {
		t.Errorf("ADDR should win over PORT, got %q", cfg.Addr)
	}
}

func TestNewClientIDIsUnique(t *testing.T) {
	seen := make(map[string]struct{}, 100000)
	for i := 0; i < 100000; i++ {
		id := newClientID()
		if _, dup := seen[id]; dup {
			t.Fatalf("duplicate ID %q after %d generations", id, i)
		}
		seen[id] = struct{}{}
	}
}

func TestSessionSignerRejectsTampering(t *testing.T) {
	s := sessionSigner{secret: []byte("k")}
	tok := s.sign("c-1")
	if !s.verify("c-1", tok) {
		t.Fatal("valid token rejected")
	}
	if s.verify("c-2", tok) {
		t.Fatal("token for another ID accepted")
	}
	if (sessionSigner{secret: []byte("other")}).verify("c-1", tok) {
		t.Fatal("token signed with another secret accepted")
	}
	if s.verify("c-1", "") {
		t.Fatal("empty token accepted")
	}
}

func TestLoadConfigRejectsBadRateLimit(t *testing.T) {
	for _, v := range []string{"abc", "-1"} {
		if _, err := LoadConfig(env(map[string]string{"AUTH_TOKEN": "x", "RATE_LIMIT": v})); err == nil {
			t.Errorf("RATE_LIMIT=%q should be rejected", v)
		}
	}
}
