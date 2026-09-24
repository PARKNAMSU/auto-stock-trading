package config_test

import (
	"strings"
	"testing"
	"time"

	"auto-stock-trading/internal/config"
)

func clearMongoEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{"APP_ENV", "MONGODB_URI", "MONGODB_CONNECT_TIMEOUT", "MONGODB_OPERATION_TIMEOUT", "MONGODB_SHUTDOWN_TIMEOUT", "MONGODB_MIN_POOL_SIZE", "MONGODB_MAX_POOL_SIZE"} {
		t.Setenv(key, "")
	}
}

func TestMongoDBEnvironmentDefaults(t *testing.T) {
	for _, tc := range []struct{ env, uri string }{
		{"", "mongodb://127.0.0.1:27017"},
		{"development", "mongodb://127.0.0.1:27017"},
		{"test", "mongodb://127.0.0.1:27018"},
	} {
		t.Run(tc.env, func(t *testing.T) {
			clearMongoEnv(t)
			t.Setenv("APP_ENV", tc.env)
			cfg, err := config.Load()
			if err != nil {
				t.Fatal(err)
			}
			if cfg.MongoDB.URI != tc.uri || cfg.MongoDB.ConnectTimeout != 5*time.Second || cfg.MongoDB.OperationTimeout != 5*time.Second || cfg.MongoDB.ShutdownTimeout != 5*time.Second || cfg.MongoDB.MinPoolSize != 0 || cfg.MongoDB.MaxPoolSize != 20 {
				t.Fatal("unexpected MongoDB defaults")
			}
		})
	}
}

func TestMongoDBProductionOverrides(t *testing.T) {
	clearMongoEnv(t)
	t.Setenv("APP_ENV", "production")
	t.Setenv("MONGODB_URI", "mongodb://db.example.test")
	t.Setenv("MONGODB_CONNECT_TIMEOUT", "2s")
	t.Setenv("MONGODB_OPERATION_TIMEOUT", "3s")
	t.Setenv("MONGODB_SHUTDOWN_TIMEOUT", "4s")
	t.Setenv("MONGODB_MIN_POOL_SIZE", "2")
	t.Setenv("MONGODB_MAX_POOL_SIZE", "10")
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.MongoDB.URI != "mongodb://db.example.test" || cfg.MongoDB.ConnectTimeout != 2*time.Second || cfg.MongoDB.OperationTimeout != 3*time.Second || cfg.MongoDB.ShutdownTimeout != 4*time.Second || cfg.MongoDB.MinPoolSize != 2 || cfg.MongoDB.MaxPoolSize != 10 {
		t.Fatal("MongoDB overrides not applied")
	}
}

func TestMongoDBInvalidSettingsDoNotExposeValues(t *testing.T) {
	for _, tc := range []struct{ key, value, field string }{
		{"APP_ENV", "production", "MONGODB_URI"},
		{"APP_ENV", "invalid-sensitive-value", "APP_ENV"},
		{"MONGODB_URI", "invalid-sensitive-value", "MONGODB_URI"},
		{"MONGODB_CONNECT_TIMEOUT", "invalid-sensitive-value", "MONGODB_CONNECT_TIMEOUT"},
		{"MONGODB_OPERATION_TIMEOUT", "0s", "MONGODB_OPERATION_TIMEOUT"},
		{"MONGODB_SHUTDOWN_TIMEOUT", "-1s", "MONGODB_SHUTDOWN_TIMEOUT"},
		{"MONGODB_MIN_POOL_SIZE", "-1", "MONGODB_MIN_POOL_SIZE"},
		{"MONGODB_MIN_POOL_SIZE", "21", "MONGODB_MIN_POOL_SIZE"},
		{"MONGODB_MAX_POOL_SIZE", "0", "MONGODB_MAX_POOL_SIZE"},
		{"MONGODB_MAX_POOL_SIZE", "18446744073709551616", "MONGODB_MAX_POOL_SIZE"},
	} {
		t.Run(tc.key+"/"+tc.value, func(t *testing.T) {
			clearMongoEnv(t)
			t.Setenv(tc.key, tc.value)
			_, err := config.Load()
			if err == nil || !strings.Contains(err.Error(), tc.field) {
				t.Fatalf("expected field error for %s", tc.field)
			}
			if strings.Contains(err.Error(), "invalid-sensitive-value") {
				t.Fatal("configuration value leaked")
			}
		})
	}
}
