package config

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	AppName                string
	AppEnv                 string
	AppPort                string
	BaseURL                string
	FrontendURL            string
	DatabaseURL            string
	ReadDatabaseURL        string
	PostgresMaxConns       int32
	PostgresMinConns       int32
	RedisAddr              string
	RedisPassword          string
	RedisDB                int
	RedisClusterAddrs      []string
	RabbitMQURL            string
	JWTAccessSecret        string
	JWTRefreshSecret       string
	JWTAccessTTL           time.Duration
	JWTRefreshTTL          time.Duration
	PasswordPepper         string
	RequestSigningSecret   string
	RequestSigningRequired bool
	EncryptionKey          string
	BcryptCost             int
	RateLimitMax           int
	RateLimitWindow        time.Duration
	LoginRateLimitMax      int
	CORSAllowedOrigins     []string
	CookieDomain           string
	CSRFSecret             string
	IPWhitelist            []string
	S3Endpoint             string
	S3AccessKey            string
	S3SecretKey            string
	S3Bucket               string
	S3Region               string
	S3UseSSL               bool
	SentryDSN              string
	PrometheusEnabled      bool
	LokiURL                string
	TracingEndpoint        string
	BackupBucket           string
	BackupRetentionDays    int
	WALArchiveEnabled      bool
	MaxConcurrentUsers     int
}

func Load() Config {
	_ = godotenv.Load("../.env", ".env")
	return Config{AppName: env("APP_NAME", "Enterprise CBT SaaS Kampus"), AppEnv: env("APP_ENV", "development"), AppPort: env("APP_PORT", "8080"), BaseURL: env("APP_BASE_URL", "http://localhost:8080"), FrontendURL: env("FRONTEND_URL", "http://localhost:5173"), DatabaseURL: env("DATABASE_URL", "postgres://cbt:cbt_password@localhost:5432/cbt?sslmode=disable"), ReadDatabaseURL: env("POSTGRES_READ_DATABASE_URL", ""), PostgresMaxConns: int32(envInt("POSTGRES_MAX_CONNS", 40)), PostgresMinConns: int32(envInt("POSTGRES_MIN_CONNS", 5)), RedisAddr: env("REDIS_ADDR", "localhost:6379"), RedisPassword: env("REDIS_PASSWORD", ""), RedisDB: envInt("REDIS_DB", 0), RedisClusterAddrs: envCSV("REDIS_CLUSTER_ADDRS", ""), RabbitMQURL: env("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"), JWTAccessSecret: env("JWT_ACCESS_SECRET", "dev-access-secret"), JWTRefreshSecret: env("JWT_REFRESH_SECRET", "dev-refresh-secret"), JWTAccessTTL: time.Duration(envInt("JWT_ACCESS_TTL_MINUTES", 15)) * time.Minute, JWTRefreshTTL: time.Duration(envInt("JWT_REFRESH_TTL_HOURS", 720)) * time.Hour, PasswordPepper: env("PASSWORD_PEPPER", "dev-pepper"), RequestSigningSecret: env("REQUEST_SIGNING_SECRET", "dev-signing"), RequestSigningRequired: envBool("REQUEST_SIGNING_REQUIRED", false), EncryptionKey: env("ENCRYPTION_KEY", "dev-key"), BcryptCost: envInt("BCRYPT_COST", 12), RateLimitMax: envInt("RATE_LIMIT_MAX", 120), RateLimitWindow: time.Duration(envInt("RATE_LIMIT_WINDOW_SECONDS", 60)) * time.Second, LoginRateLimitMax: envInt("LOGIN_RATE_LIMIT_MAX", 10), CORSAllowedOrigins: envCSV("CORS_ALLOWED_ORIGINS", "http://localhost:5173"), CookieDomain: env("COOKIE_DOMAIN", ""), CSRFSecret: env("CSRF_SECRET", "dev-csrf"), IPWhitelist: envCSV("IP_WHITELIST", ""), S3Endpoint: env("S3_ENDPOINT", "localhost:9000"), S3AccessKey: env("S3_ACCESS_KEY", "minioadmin"), S3SecretKey: env("S3_SECRET_KEY", "minioadmin"), S3Bucket: env("S3_BUCKET", "cbt-assets"), S3Region: env("S3_REGION", "us-east-1"), S3UseSSL: envBool("S3_USE_SSL", false), SentryDSN: env("SENTRY_DSN", ""), PrometheusEnabled: envBool("PROMETHEUS_ENABLED", true), LokiURL: env("LOKI_URL", ""), TracingEndpoint: env("TRACING_ENDPOINT", ""), BackupBucket: env("BACKUP_BUCKET", "cbt-backups"), BackupRetentionDays: envInt("BACKUP_RETENTION_DAYS", 30), WALArchiveEnabled: envBool("WAL_ARCHIVE_ENABLED", true), MaxConcurrentUsers: envInt("MAX_CONCURRENT_USERS", 100000)}
}
func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
func envInt(k string, d int) int {
	v := env(k, "")
	if v == "" {
		return d
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return d
	}
	return n
}
func envBool(k string, d bool) bool {
	v := strings.ToLower(env(k, ""))
	if v == "" {
		return d
	}
	return v == "true" || v == "1" || v == "yes"
}
func envCSV(k, d string) []string {
	raw := env(k, d)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if s := strings.TrimSpace(p); s != "" {
			out = append(out, s)
		}
	}
	return out
}
