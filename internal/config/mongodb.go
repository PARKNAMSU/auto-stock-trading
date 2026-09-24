package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"auto-stock-trading/internal/mongodb"
)

func loadMongoDB(environment string) (mongodb.Config, error) {
	var defaultURI string
	switch environment {
	case "development":
		defaultURI = "mongodb://127.0.0.1:27017"
	case "test":
		defaultURI = "mongodb://127.0.0.1:27018"
	case "production":
	default:
		return mongodb.Config{}, fmt.Errorf("APP_ENV must be development, test or production")
	}
	cfg := mongodb.Config{
		URI:            valueOrDefault("MONGODB_URI", defaultURI),
		ConnectTimeout: 5 * time.Second, OperationTimeout: 5 * time.Second,
		ShutdownTimeout: 5 * time.Second, MaxPoolSize: 20,
	}
	if cfg.URI == "" {
		return mongodb.Config{}, fmt.Errorf("MONGODB_URI is required in production")
	}
	for _, setting := range []struct {
		key    string
		target *time.Duration
	}{
		{"MONGODB_CONNECT_TIMEOUT", &cfg.ConnectTimeout},
		{"MONGODB_OPERATION_TIMEOUT", &cfg.OperationTimeout},
		{"MONGODB_SHUTDOWN_TIMEOUT", &cfg.ShutdownTimeout},
	} {
		if raw := os.Getenv(setting.key); raw != "" {
			value, err := time.ParseDuration(raw)
			if err != nil || value <= 0 {
				return mongodb.Config{}, fmt.Errorf("%s must be a positive duration (for example 5s)", setting.key)
			}
			*setting.target = value
		}
	}
	for _, setting := range []struct {
		key    string
		target *uint64
	}{
		{"MONGODB_MIN_POOL_SIZE", &cfg.MinPoolSize},
		{"MONGODB_MAX_POOL_SIZE", &cfg.MaxPoolSize},
	} {
		if raw := os.Getenv(setting.key); raw != "" {
			value, err := strconv.ParseUint(raw, 10, 64)
			if err != nil {
				return mongodb.Config{}, fmt.Errorf("%s must be a non-negative integer", setting.key)
			}
			*setting.target = value
		}
	}
	if err := cfg.Validate(); err != nil {
		return mongodb.Config{}, err
	}
	return cfg, nil
}
