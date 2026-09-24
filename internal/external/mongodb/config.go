package mongodb

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	URI              string `json:"-"`
	ConnectTimeout   time.Duration
	OperationTimeout time.Duration
	ShutdownTimeout  time.Duration
	MinPoolSize      uint64
	MaxPoolSize      uint64
}

// 설정 전체를 포맷하거나 구조화 로그로 출력해도 URI는 노출하지 않습니다.
func (Config) String() string         { return "MongoDB configuration (URI redacted)" }
func (c Config) GoString() string     { return c.String() }
func (c Config) LogValue() slog.Value { return slog.StringValue(c.String()) }

func (c Config) Validate() error {
	if !strings.HasPrefix(c.URI, "mongodb://") && !strings.HasPrefix(c.URI, "mongodb+srv://") {
		return errors.New("MONGODB_URI must use mongodb:// or mongodb+srv://")
	}
	if c.ConnectTimeout <= 0 || c.OperationTimeout <= 0 || c.ShutdownTimeout <= 0 {
		return errors.New("MongoDB timeouts must be positive")
	}
	if c.MaxPoolSize == 0 || c.MinPoolSize > c.MaxPoolSize {
		return errors.New("MongoDB pool requires 0 <= MONGODB_MIN_POOL_SIZE <= MONGODB_MAX_POOL_SIZE and a positive maximum")
	}
	return nil
}

// LoadConfig는 호출자가 결정한 실행 환경에 맞춰 MONGODB_* 환경변수를 읽고 검증합니다.
// environment는 development, test 또는 production이어야 하며 APP_ENV는 직접 읽지 않습니다.
func LoadConfig(environment string) (Config, error) {
	var defaultURI string
	switch environment {
	case "development":
		defaultURI = "mongodb://127.0.0.1:27017"
	case "test":
		defaultURI = "mongodb://127.0.0.1:27018"
	case "production":
	default:
		return Config{}, fmt.Errorf("APP_ENV must be development, test or production")
	}
	cfg := Config{
		URI:            os.Getenv("MONGODB_URI"),
		ConnectTimeout: 5 * time.Second, OperationTimeout: 5 * time.Second,
		ShutdownTimeout: 5 * time.Second, MaxPoolSize: 20,
	}
	if cfg.URI == "" {
		cfg.URI = defaultURI
	}
	if cfg.URI == "" {
		return Config{}, fmt.Errorf("MONGODB_URI is required in production")
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
				return Config{}, fmt.Errorf("%s must be a positive duration (for example 5s)", setting.key)
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
				return Config{}, fmt.Errorf("%s must be a non-negative integer", setting.key)
			}
			*setting.target = value
		}
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}
