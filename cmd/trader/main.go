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
	"auto-stock-trading/internal/marketdata"
	"auto-stock-trading/internal/risk"
	"auto-stock-trading/internal/strategy"
	"auto-stock-trading/internal/tossinvest"
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

	market := domain.Market(cfg.TradingMarket)
	apiClient, err := tossinvest.NewClient(tossinvest.Config{
		ClientID: cfg.ClientID, ClientSecret: cfg.ClientSecret,
		BaseURL: cfg.TossBaseURL, Logger: logger,
	})
	if err != nil {
		logger.Error("invalid Toss Securities API configuration", "error", err)
		os.Exit(1)
	}
	collector, err := marketdata.NewCollector(apiClient, marketdata.Config{})
	if err != nil {
		logger.Error("create market data collector", "error", err)
		os.Exit(1)
	}
	selectedStrategy, err := strategy.NewScoreEngineForMarket(market)
	if err != nil {
		logger.Error("invalid market strategy", "error", err)
		os.Exit(1)
	}

	engine := trading.NewEngine(
		selectedStrategy,
		risk.NewManager(cfg.MaxOrderAmount),
		trading.NewDryRunExecutor(logger),
		logger,
	)

	logger.Info("trader started", "mode", cfg.TradingMode, "market", market, "strategy", "score-engine")
	snapshots, err := collector.Snapshots(ctx, market)
	if err != nil {
		logger.Error("collect market data", "error", err)
		os.Exit(1)
	}
	for _, snapshot := range snapshots {
		if snapshot.IsStale || len(snapshot.MissingFields) > 0 {
			logger.Warn("market snapshot quality warning", "symbol", snapshot.Symbol, "stale", snapshot.IsStale, "missing_fields", snapshot.MissingFields)
		}
		if err := engine.RunOnce(ctx, snapshot); err != nil {
			logger.Error("trading cycle failed", "symbol", snapshot.Symbol, "error", err)
			os.Exit(1)
		}
	}
}
