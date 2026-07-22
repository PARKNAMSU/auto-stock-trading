// trader 애플리케이션을 구성하고 한 번의 거래 사이클을 실행하는 진입점입니다.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"auto-stock-trading/internal/config"
	"auto-stock-trading/internal/domain"
	"auto-stock-trading/internal/risk"
	"auto-stock-trading/internal/strategy"
	"auto-stock-trading/internal/trading"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg, err := config.Load()
	if err != nil {
		logger.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	selectedStrategy, err := strategy.New(strategy.Kind(cfg.TradingStrategy))
	if err != nil {
		logger.Error("invalid strategy", "error", err)
		os.Exit(1)
	}

	engine := trading.NewEngine(
		selectedStrategy,
		risk.NewManager(cfg.MaxOrderAmount),
		trading.NewDryRunExecutor(logger),
		logger,
	)

	logger.Info("trader started", "mode", cfg.TradingMode, "strategy", cfg.TradingStrategy)
	if err := engine.RunOnce(ctx, domain.MarketSnapshot{}); err != nil {
		logger.Error("trading cycle failed", "error", err)
		os.Exit(1)
	}
}
