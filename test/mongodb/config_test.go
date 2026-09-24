package mongodb_test

import (
	"strings"
	"testing"
	"time"

	"auto-stock-trading/internal/external/mongodb"
)

func clearMongoEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{"APP_ENV", "MONGODB_URI", "MONGODB_CONNECT_TIMEOUT", "MONGODB_OPERATION_TIMEOUT", "MONGODB_SHUTDOWN_TIMEOUT", "MONGODB_MIN_POOL_SIZE", "MONGODB_MAX_POOL_SIZE"} {
		t.Setenv(key, "")
	}
}

func TestMongoDBEnvironmentDefaults(t *testing.T) {
	for _, tc := range []struct{ env, uri string }{
		{"development", "mongodb://127.0.0.1:27017"},
		{"test", "mongodb://127.0.0.1:27018"},
	} {
		t.Run(tc.env, func(t *testing.T) {
			clearMongoEnv(t)
			// 실제 APP_ENV와 무관하게 전달받은 환경으로 설정해야 합니다.
			t.Setenv("APP_ENV", "production")
			cfg, err := mongodb.LoadConfig(tc.env)
			if err != nil {
				t.Fatal(err)
			}
			if cfg.URI != tc.uri || cfg.ConnectTimeout != 5*time.Second || cfg.OperationTimeout != 5*time.Second || cfg.ShutdownTimeout != 5*time.Second || cfg.MinPoolSize != 0 || cfg.MaxPoolSize != 20 {
				t.Fatal("unexpected MongoDB defaults")
			}
		})
	}
}

func TestMongoDBProductionOverrides(t *testing.T) {
	clearMongoEnv(t)
	t.Setenv("MONGODB_URI", "mongodb://db.example.test")
	t.Setenv("MONGODB_CONNECT_TIMEOUT", "2s")
	t.Setenv("MONGODB_OPERATION_TIMEOUT", "3s")
	t.Setenv("MONGODB_SHUTDOWN_TIMEOUT", "4s")
	t.Setenv("MONGODB_MIN_POOL_SIZE", "2")
	t.Setenv("MONGODB_MAX_POOL_SIZE", "10")
	cfg, err := mongodb.LoadConfig("production")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.URI != "mongodb://db.example.test" || cfg.ConnectTimeout != 2*time.Second || cfg.OperationTimeout != 3*time.Second || cfg.ShutdownTimeout != 4*time.Second || cfg.MinPoolSize != 2 || cfg.MaxPoolSize != 10 {
		t.Fatal("MongoDB overrides not applied")
	}
}

func TestMongoDBInvalidSettingsDoNotExposeValues(t *testing.T) {
	for _, tc := range []struct{ key, value, field string }{
		{"APP_ENV", "production", "MONGODB_URI"},
		{"APP_ENV", "invalid-sensitive-value", "APP_ENV"},
		{"APP_ENV", "", "APP_ENV"},
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
			environment := "development"
			if tc.key == "APP_ENV" {
				environment = tc.value
			}
			_, err := mongodb.LoadConfig(environment)
			if err == nil || !strings.Contains(err.Error(), tc.field) {
				t.Fatalf("expected field error for %s", tc.field)
			}
			if strings.Contains(err.Error(), "invalid-sensitive-value") {
				t.Fatal("configuration value leaked")
			}
		})
	}
}
