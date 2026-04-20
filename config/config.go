package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// Config holds all configuration for the service.
// To add a new upstream service later: add a struct block here and its env vars in .env.
// Endpoint paths are always hardcoded here — never in .env.
type Config struct {
	Port          string
	XAPIKey string

	// Redis — only connected when CacheEnabled is true
	RedisAddr    string
	CacheEnabled bool

	// Cache TTLs per section — configurable per environment
	CacheTTL struct {
		Disbursement time.Duration
		Repayment    time.Duration
		Mandate      time.Duration
		Payments     time.Duration
		Kyc          time.Duration
		Application  time.Duration
		Refund       time.Duration
		Documents    time.Duration
	}

	// Admin service (gqadmin_backend)
	Admin struct {
		BaseURL string
		APIKey  string
		// All admin endpoint paths — add new ones here as you integrate more APIs
		Endpoints struct {
			Disbursement       string
			ApplicationSummary string // template: fmt.Sprintf(path, appID)
			Refunds            string
			Payments           string
		}
	}

	// Repayment service
	Repayment struct {
		BaseURL string
		APIKey  string
		Endpoints struct {
			Schedule string
		}
	}

	// KYC service (svc-kyc)
	Kyc struct {
		BaseURL string
		APIKey  string
		// All KYC endpoint paths
		Endpoints struct {
			Status string // template: fmt.Sprintf(path, appID)
		}
	}
}

// Load reads all config from environment variables or a .env file.
// Call once at startup in main.go, then pass the *Config around.
func Load() (*Config, error) {
	viper.SetConfigFile(".env")
	viper.SetConfigType("env")
	viper.AutomaticEnv() // env vars take precedence over .env file (important for Docker/K8s)

	// Missing .env file is fine — env vars still work
	_ = viper.ReadInConfig()

	cfg := &Config{}

	// ── Server ────────────────────────────────────────────────────────────────
	cfg.Port = viper.GetString("PORT")
	if cfg.Port == "" {
		cfg.Port = "8080"
	}

	cfg.XAPIKey = viper.GetString("X_API_KEY")
	if cfg.XAPIKey == "" {
		return nil, fmt.Errorf("X_API_KEY is required but not set")
	}

	// ── Redis ─────────────────────────────────────────────────────────────────
	cfg.RedisAddr = viper.GetString("REDIS_ADDR")
	cfg.CacheEnabled = viper.GetBool("CACHE_ENABLED")

	// ── Cache TTLs ────────────────────────────────────────────────────────────
	cfg.CacheTTL.Disbursement = viper.GetDuration("CACHE_TTL_DISBURSEMENT")
	if cfg.CacheTTL.Disbursement == 0 {
		cfg.CacheTTL.Disbursement = 2 * time.Hour
	}
	cfg.CacheTTL.Repayment = viper.GetDuration("CACHE_TTL_REPAYMENT")
	if cfg.CacheTTL.Repayment == 0 {
		cfg.CacheTTL.Repayment = 30 * time.Minute
	}
	cfg.CacheTTL.Mandate = viper.GetDuration("CACHE_TTL_MANDATE")
	if cfg.CacheTTL.Mandate == 0 {
		cfg.CacheTTL.Mandate = 15 * time.Minute
	}
	cfg.CacheTTL.Payments = viper.GetDuration("CACHE_TTL_PAYMENTS")
	if cfg.CacheTTL.Payments == 0 {
		cfg.CacheTTL.Payments = 5 * time.Minute
	}
	cfg.CacheTTL.Kyc = viper.GetDuration("CACHE_TTL_KYC")
	if cfg.CacheTTL.Kyc == 0 {
		cfg.CacheTTL.Kyc = 1 * time.Hour
	}
	cfg.CacheTTL.Application = viper.GetDuration("CACHE_TTL_APPLICATION")
	if cfg.CacheTTL.Application == 0 {
		cfg.CacheTTL.Application = 5 * time.Minute
	}
	cfg.CacheTTL.Refund = viper.GetDuration("CACHE_TTL_REFUND")
	if cfg.CacheTTL.Refund == 0 {
		cfg.CacheTTL.Refund = 15 * time.Minute
	}
	cfg.CacheTTL.Documents = viper.GetDuration("CACHE_TTL_DOCUMENTS")
	if cfg.CacheTTL.Documents == 0 {
		cfg.CacheTTL.Documents = 1 * time.Hour
	}

	// ── Admin service ─────────────────────────────────────────────────────────
	cfg.Admin.BaseURL = viper.GetString("ADMIN_BASE_URL")
	if cfg.Admin.BaseURL == "" {
		return nil, fmt.Errorf("ADMIN_BASE_URL is required but not set")
	}
	cfg.Admin.APIKey = viper.GetString("ADMIN_API_KEY")
	if cfg.Admin.APIKey == "" {
		return nil, fmt.Errorf("ADMIN_API_KEY is required but not set")
	}
	// Endpoint paths hardcoded here so they're all in one place
	cfg.Admin.Endpoints.Disbursement = "/v0.1/disbursal-requests/fetch"
	cfg.Admin.Endpoints.ApplicationSummary = "/v0.1/applications/summary/%s/fetch"
	cfg.Admin.Endpoints.Refunds = "/v1/refunds/requests/fetch"
	cfg.Admin.Endpoints.Payments = "/v1/payments/transactions/fetch"

	// ── Repayment service ─────────────────────────────────────────────────────
	cfg.Repayment.BaseURL = viper.GetString("SERVICES_BASE_URL")
	if cfg.Repayment.BaseURL == "" {
		return nil, fmt.Errorf("SERVICES_BASE_URL is required but not set")
	}
	cfg.Repayment.APIKey = viper.GetString("SERVICES_API_KEY")
	if cfg.Repayment.APIKey == "" {
		return nil, fmt.Errorf("SERVICES_API_KEY is required but not set")
	}
	cfg.Repayment.Endpoints.Schedule = "/v1/repayment-schedule/fetch"

	// ── KYC service ───────────────────────────────────────────────────────────
	cfg.Kyc.BaseURL = viper.GetString("KYC_BASE_URL")
	if cfg.Kyc.BaseURL == "" {
		return nil, fmt.Errorf("KYC_BASE_URL is required but not set")
	}
	cfg.Kyc.APIKey = viper.GetString("KYC_API_KEY")
	if cfg.Kyc.APIKey == "" {
		return nil, fmt.Errorf("KYC_API_KEY is required but not set")
	}
	cfg.Kyc.Endpoints.Status = "/v1/misc/kyc/%s/status"

	return cfg, nil
}