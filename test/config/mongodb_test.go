package config_test

import (
	"strings"
	"testing"

	"auto-stock-trading/internal/config"
)

// 애플리케이션이 APP_ENV를 전달하고 MongoDB 설정 및 오류를 조합하는지 확인합니다.
func TestLoadComposesMongoDBConfig(t *testing.T) {
	for _, tc := range []struct {
		name, environment, uri, timeout, wantURI, errorField string
	}{
		{name: "default", wantURI: "mongodb://127.0.0.1:27017"},
		{name: "test", environment: "test", wantURI: "mongodb://127.0.0.1:27018"},
		{name: "production", environment: "production", uri: "mongodb://db.example.test", wantURI: "mongodb://db.example.test"},
		{name: "missing production URI", environment: "production", errorField: "MONGODB_URI"},
		{name: "invalid timeout", timeout: "0s", errorField: "MONGODB_CONNECT_TIMEOUT"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("APP_ENV", tc.environment)
			t.Setenv("MONGODB_URI", tc.uri)
			t.Setenv("MONGODB_CONNECT_TIMEOUT", tc.timeout)
			for _, key := range []string{"MONGODB_OPERATION_TIMEOUT", "MONGODB_SHUTDOWN_TIMEOUT", "MONGODB_MIN_POOL_SIZE", "MONGODB_MAX_POOL_SIZE"} {
				t.Setenv(key, "")
			}
			cfg, err := config.Load()
			if tc.errorField != "" {
				if err == nil || !strings.Contains(err.Error(), tc.errorField) {
					t.Fatalf("expected error for %s", tc.errorField)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if cfg.MongoDB.URI != tc.wantURI {
				t.Fatal("MongoDB settings do not match application environment")
			}
		})
	}
}
