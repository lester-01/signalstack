package main

import (
	"crypto/rand"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Config holds everything that varies between environments.
type Config struct {
	// Addr is the listen address, e.g. ":8080".
	Addr string
	// AuthToken is the shared secret clients must present in their auth message.
	AuthToken string
	// AllowedOrigins lists browser origins permitted to open a WebSocket.
	// "*" allows any origin. Requests without an Origin header (non-browser
	// clients) are always allowed; they still have to authenticate.
	AllowedOrigins []string
	// SessionSecret signs the resume tokens that let a client reclaim its ID.
	SessionSecret []byte
	// RateLimit is the sustained chat messages per second allowed per client
	// (burst is twice that). 0 disables limiting.
	RateLimit float64
	// MetricsEnabled exposes /metrics. It reveals runtime details, so keep it
	// off public interfaces.
	MetricsEnabled bool
	// SessionSecretGenerated is true when no SESSION_SECRET was provided and a
	// random one was created, so sessions will not survive a restart.
	SessionSecretGenerated bool
}

// LoadConfig builds a Config from environment variables:
//
//	AUTH_TOKEN       required, shared token clients must present
//	ADDR             listen address (default ":8080"; PORT is used if ADDR is unset)
//	ALLOWED_ORIGINS  comma-separated origins (default "http://localhost:3000")
//	SESSION_SECRET   key for signing resume tokens (default: random per process)
//	RATE_LIMIT       chat messages/second per client, 0 to disable (default 10)
//	METRICS_ENABLED  "true" to serve /metrics (default false)
func LoadConfig(getenv func(string) string) (Config, error) {
	cfg := Config{AuthToken: getenv("AUTH_TOKEN")}
	if cfg.AuthToken == "" {
		return Config{}, errors.New("AUTH_TOKEN is required")
	}

	switch {
	case getenv("ADDR") != "":
		cfg.Addr = getenv("ADDR")
	case getenv("PORT") != "":
		cfg.Addr = ":" + getenv("PORT")
	default:
		cfg.Addr = ":8080"
	}

	origins := getenv("ALLOWED_ORIGINS")
	if origins == "" {
		origins = "http://localhost:3000"
	}
	for _, o := range strings.Split(origins, ",") {
		if o = strings.TrimSpace(o); o != "" {
			cfg.AllowedOrigins = append(cfg.AllowedOrigins, o)
		}
	}

	cfg.RateLimit = 10
	if v := getenv("RATE_LIMIT"); v != "" {
		f, err := strconv.ParseFloat(v, 64)
		if err != nil || f < 0 {
			return Config{}, fmt.Errorf("RATE_LIMIT must be a non-negative number, got %q", v)
		}
		cfg.RateLimit = f
	}
	cfg.MetricsEnabled, _ = strconv.ParseBool(getenv("METRICS_ENABLED"))

	if s := getenv("SESSION_SECRET"); s != "" {
		cfg.SessionSecret = []byte(s)
	} else {
		cfg.SessionSecret = make([]byte, 32)
		if _, err := rand.Read(cfg.SessionSecret); err != nil {
			return Config{}, fmt.Errorf("generate session secret: %w", err)
		}
		cfg.SessionSecretGenerated = true
	}

	return cfg, nil
}
