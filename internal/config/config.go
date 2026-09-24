// Package config는 환경 변수에서 애플리케이션 설정을 읽고 검증합니다.
package config

import (
	"fmt"
	"os"
	"strconv"

	"auto-stock-trading/internal/mongodb"
)

const (
	DryRunMode    = "dry-run"
	DefaultMarket = "us"
	USMarket      = "us"
	KRMarket      = "kr"
)

type Config struct {
	Environment    string
	MongoDB        mongodb.Config
	TradingMode    string
	TradingMarket  string
	ClientID       string
	ClientSecret   string
	Account        string
	TossBaseURL    string
	MaxOrderAmount int64
}

func Load() (Config, error) {
	cfg := Config{
		Environment:    valueOrDefault("APP_ENV", "development"),
		TradingMode:    valueOrDefault("TRADING_MODE", DryRunMode),
		TradingMarket:  valueOrDefault("TRADING_MARKET", DefaultMarket),
		ClientID:       os.Getenv("TOSSINVEST_CLIENT_ID"),
		ClientSecret:   os.Getenv("TOSSINVEST_CLIENT_SECRET"),
		Account:        os.Getenv("TOSSINVEST_ACCOUNT"),
		TossBaseURL:    valueOrDefault("TOSSINVEST_BASE_URL", "https://openapi.tossinvest.com"),
		MaxOrderAmount: 100_000,
	}

	if raw := os.Getenv("MAX_ORDER_AMOUNT"); raw != "" {
		amount, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || amount <= 0 {
			return Config{}, fmt.Errorf("MAX_ORDER_AMOUNT must be a positive integer")
		}
		cfg.MaxOrderAmount = amount
	}

	if cfg.TradingMode != DryRunMode {
		return Config{}, fmt.Errorf("unsupported TRADING_MODE %q; only dry-run is currently available", cfg.TradingMode)
	}
	if cfg.TradingMarket != USMarket && cfg.TradingMarket != KRMarket {
		return Config{}, fmt.Errorf("unsupported TRADING_MARKET %q; use us or kr", cfg.TradingMarket)
	}

	mongoConfig, err := loadMongoDB(cfg.Environment)
	if err != nil {
		return Config{}, err
	}
	cfg.MongoDB = mongoConfig
	return cfg, nil
}

func valueOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
