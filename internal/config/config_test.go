// 이 파일은 거래 대상 시장 환경 변수의 기본값과 유효성 검사를 검증합니다.
package config

import "testing"

func TestLoadDefaultsToUSMarket(t *testing.T) {
	t.Setenv("TRADING_MARKET", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned an error: %v", err)
	}
	if cfg.TradingMarket != USMarket {
		t.Fatalf("expected %q, got %q", USMarket, cfg.TradingMarket)
	}
}

func TestLoadAcceptsKRMarket(t *testing.T) {
	t.Setenv("TRADING_MARKET", KRMarket)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned an error: %v", err)
	}
	if cfg.TradingMarket != KRMarket {
		t.Fatalf("expected %q, got %q", KRMarket, cfg.TradingMarket)
	}
}

func TestLoadRejectsUnknownMarket(t *testing.T) {
	t.Setenv("TRADING_MARKET", "jp")

	if _, err := Load(); err == nil {
		t.Fatal("Load() accepted an unsupported market")
	}
}
