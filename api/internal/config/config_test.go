package config

import "testing"

func TestEnvHelper(t *testing.T) {
	t.Run("returns default when unset", func(t *testing.T) {
		t.Setenv("CR_TEST_KEY", "")
		if got := env("CR_TEST_KEY", "fallback"); got != "fallback" {
			t.Errorf("env unset = %q, want fallback", got)
		}
	})
	t.Run("trims whitespace and returns default for blank", func(t *testing.T) {
		t.Setenv("CR_TEST_KEY", "   ")
		if got := env("CR_TEST_KEY", "fallback"); got != "fallback" {
			t.Errorf("env blank = %q, want fallback", got)
		}
	})
	t.Run("returns value when set", func(t *testing.T) {
		t.Setenv("CR_TEST_KEY", "  hi  ")
		if got := env("CR_TEST_KEY", "fallback"); got != "hi" {
			t.Errorf("env set = %q, want hi", got)
		}
	})
}

func TestLoadDefaults(t *testing.T) {
	// Clear the relevant vars so defaults are exercised.
	for _, k := range []string{
		"HTTP_PORT", "JOB_TIMEOUT_SECONDS", "SESSION_TTL_HOURS",
		"ALLOW_REGISTRATION", "JWT_SECRET", "LOG_LEVEL",
	} {
		t.Setenv(k, "")
	}
	cfg := Load()
	if cfg.HTTPPort != "5488" {
		t.Errorf("HTTPPort = %q, want 5488", cfg.HTTPPort)
	}
	if cfg.JobTimeoutSeconds != 120 {
		t.Errorf("JobTimeoutSeconds = %d, want 120", cfg.JobTimeoutSeconds)
	}
	if cfg.SessionTTLHours != 720 {
		t.Errorf("SessionTTLHours = %d, want 720", cfg.SessionTTLHours)
	}
	if !cfg.AllowRegistration {
		t.Errorf("AllowRegistration default = %v, want true", cfg.AllowRegistration)
	}
	if cfg.JWTSecret != "change-me" {
		t.Errorf("JWTSecret = %q, want change-me", cfg.JWTSecret)
	}
}

func TestLoadSessionTTL(t *testing.T) {
	tests := []struct {
		name string
		val  string
		want int
	}{
		{"parses positive", "48", 48},
		{"zero falls back", "0", 720},
		{"negative falls back", "-5", 720},
		{"non-numeric falls back to 720", "abc", 720},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("SESSION_TTL_HOURS", tt.val)
			cfg := Load()
			if cfg.SessionTTLHours != tt.want {
				t.Errorf("SessionTTLHours = %d, want %d", cfg.SessionTTLHours, tt.want)
			}
		})
	}
}

func TestLoadAllowRegistration(t *testing.T) {
	tests := []struct {
		val  string
		want bool
	}{
		{"true", true},
		{"false", false},
		{"yes", false},  // only literal "true" enables it
		{"1", false},
		{"TRUE", false}, // case-sensitive
	}
	for _, tt := range tests {
		t.Run(tt.val, func(t *testing.T) {
			t.Setenv("ALLOW_REGISTRATION", tt.val)
			cfg := Load()
			if cfg.AllowRegistration != tt.want {
				t.Errorf("ALLOW_REGISTRATION=%q -> %v, want %v", tt.val, cfg.AllowRegistration, tt.want)
			}
		})
	}
}

func TestLoadJobTimeout(t *testing.T) {
	t.Setenv("JOB_TIMEOUT_SECONDS", "300")
	if cfg := Load(); cfg.JobTimeoutSeconds != 300 {
		t.Errorf("JobTimeoutSeconds = %d, want 300", cfg.JobTimeoutSeconds)
	}
	// Non-numeric -> Atoi error -> 0 (no fallback for job timeout).
	t.Setenv("JOB_TIMEOUT_SECONDS", "oops")
	if cfg := Load(); cfg.JobTimeoutSeconds != 0 {
		t.Errorf("JobTimeoutSeconds bad = %d, want 0", cfg.JobTimeoutSeconds)
	}
}
