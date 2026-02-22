package config

import (
	"os"
	"strconv"
	"strings"
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
	EdgeDeviceWhitelist    []string
	RateLimitPerUserMin    int
	RateLimitPerIPMin      int
	AdminToken             string
	WhitelistEnabled       bool
	WhitelistUsers         []string
	WhitelistDailyQuota    int
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
		EdgeDeviceWhitelist:    getEnvList("EDGE_DEVICE_WHITELIST", []string{}),
		RateLimitPerUserMin:    getEnvInt("RATE_LIMIT_PER_USER_MIN", 120),
		RateLimitPerIPMin:      getEnvInt("RATE_LIMIT_PER_IP_MIN", 300),
		AdminToken:             getEnv("ADMIN_TOKEN", "dev-admin-token"),
		WhitelistEnabled:       getEnvBool("WHITELIST_ENABLED", false),
		WhitelistUsers:         getEnvList("WHITELIST_USERS", []string{}),
		WhitelistDailyQuota:    getEnvInt("WHITELIST_DAILY_QUOTA", 100),
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

func getEnvList(key string, defaultValue []string) []string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return append([]string{}, defaultValue...)
	}
	items := strings.Split(value, ",")
	result := make([]string, 0, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	if len(result) == 0 {
		return append([]string{}, defaultValue...)
	}
	return result
}
