package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Addr                   string
	InferenceServiceURL    string
	MySQLDSN               string
	RedisAddr              string
	DefaultRetentionDays   int
	EdgeAcceptThreshold    float64
	CloudFallbackThreshold float64
	CopyTimeout            time.Duration
	ModelVersion           string
	PainRiskEnabled        bool
}

func Load() Config {
	return Config{
		Addr:                   getEnv("API_ADDR", ":8080"),
		InferenceServiceURL:    getEnv("INFERENCE_URL", "http://localhost:8081"),
		MySQLDSN:               getEnv("MYSQL_DSN", ""),
		RedisAddr:              getEnv("REDIS_ADDR", ""),
		DefaultRetentionDays:   getEnvInt("DEFAULT_RETENTION_DAYS", 7),
		EdgeAcceptThreshold:    getEnvFloat("EDGE_ACCEPT_THRESHOLD", 0.70),
		CloudFallbackThreshold: getEnvFloat("CLOUD_FALLBACK_THRESHOLD", 0.45),
		CopyTimeout:            time.Duration(getEnvInt("COPY_TIMEOUT_MS", 1200)) * time.Millisecond,
		ModelVersion:           getEnv("MODEL_VERSION", "mobilenetv3-small-int8-v1"),
		PainRiskEnabled:        getEnvBool("PAIN_RISK_ENABLED", false),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return parsed
}

func getEnvFloat(key string, defaultValue float64) float64 {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return defaultValue
	}
	return parsed
}

func getEnvBool(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return defaultValue
	}
	return parsed
}
