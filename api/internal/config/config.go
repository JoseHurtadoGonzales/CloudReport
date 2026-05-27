package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	HTTPPort          string
	DatabaseURL       string
	RedisURL          string
	JWTSecret         string
	LogLevel          string
	AllowRegistration bool
	// SessionTTLHours controls how long an issued JWT stays valid. The
	// frontend silently refreshes well before this, so a long value means
	// "stay logged in" without re-prompting. Default: 30 days.
	SessionTTLHours int

	S3Endpoint  string
	S3Region    string
	S3Bucket    string
	S3AccessKey string
	S3SecretKey string

	JobTimeoutSeconds int

	InitialAdminUsername string
	InitialAdminPassword string
	InitialAdminEmail    string
}

func env(key, def string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	return v
}

func Load() *Config {
	jobTimeout, _ := strconv.Atoi(env("JOB_TIMEOUT_SECONDS", "120"))
	sessionTTL, _ := strconv.Atoi(env("SESSION_TTL_HOURS", "720")) // 720h = 30 days
	if sessionTTL <= 0 {
		sessionTTL = 720
	}
	return &Config{
		HTTPPort:          env("HTTP_PORT", "5488"),
		DatabaseURL:       env("DATABASE_URL", "postgres://cloudreport:cloudreport@localhost:5432/cloudreport?sslmode=disable"),
		RedisURL:          env("REDIS_URL", "redis://localhost:6379/0"),
		JWTSecret:         env("JWT_SECRET", "change-me"),
		LogLevel:          env("LOG_LEVEL", "info"),
		AllowRegistration: env("ALLOW_REGISTRATION", "true") == "true",
		SessionTTLHours:   sessionTTL,

		S3Endpoint:  env("WEED_S3", "http://localhost:8333"),
		S3Region:    env("S3_REGION", "us-east-1"),
		S3Bucket:    env("S3_BUCKET", "cloudreport"),
		S3AccessKey: env("S3_ACCESS_KEY", "cloudreport"),
		S3SecretKey: env("S3_SECRET_KEY", "cloudreport123"),

		JobTimeoutSeconds: jobTimeout,

		InitialAdminUsername: env("INITIAL_ADMIN_USERNAME", ""),
		InitialAdminPassword: env("INITIAL_ADMIN_PASSWORD", ""),
		InitialAdminEmail:    env("INITIAL_ADMIN_EMAIL", ""),
	}
}
