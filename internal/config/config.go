package config

import (
	"fmt"
	"os"
	"strconv"
)

const DryRunMode = "dry-run"

type Config struct {
	TradingMode    string
	ClientID       string
	ClientSecret   string
	Account        string
	MaxOrderAmount int64
}

func Load() (Config, error) {
	cfg := Config{
		TradingMode:    valueOrDefault("TRADING_MODE", DryRunMode),
		ClientID:       os.Getenv("TOSSINVEST_CLIENT_ID"),
		ClientSecret:   os.Getenv("TOSSINVEST_CLIENT_SECRET"),
		Account:        os.Getenv("TOSSINVEST_ACCOUNT"),
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

	return cfg, nil
}

func valueOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
