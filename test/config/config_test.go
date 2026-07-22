// 이 파일은 외부 패키지 관점에서 거래 대상 시장 환경 변수의 기본값과 유효성 검사를 검증합니다.
package config_test

import (
	"testing"

	"auto-stock-trading/internal/config"
)

func TestLoadDefaultsToUSMarket(t *testing.T) {
	t.Setenv("TRADING_MARKET", "")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() returned an error: %v", err)
	}
	if cfg.TradingMarket != config.USMarket {
		t.Fatalf("expected %q, got %q", config.USMarket, cfg.TradingMarket)
	}
}

func TestLoadAcceptsKRMarket(t *testing.T) {
	t.Setenv("TRADING_MARKET", config.KRMarket)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() returned an error: %v", err)
	}
	if cfg.TradingMarket != config.KRMarket {
		t.Fatalf("expected %q, got %q", config.KRMarket, cfg.TradingMarket)
	}
}

func TestLoadRejectsUnknownMarket(t *testing.T) {
	t.Setenv("TRADING_MARKET", "jp")

	if _, err := config.Load(); err == nil {
		t.Fatal("Load() accepted an unsupported market")
	}
}
