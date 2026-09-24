// trader 애플리케이션을 구성하고 한 번의 거래 사이클을 실행하는 진입점입니다.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"auto-stock-trading/internal/config"
	"auto-stock-trading/internal/domain"
	"auto-stock-trading/internal/external/mongodb"
	"auto-stock-trading/internal/external/tossinvest"
	"auto-stock-trading/internal/marketdata"
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
	if err := run(ctx, cfg, logger); err != nil {
		logger.Error("trader stopped", "error", err)
		os.Exit(1)
	}
}

// main에서 종료 코드를 정하기 전에 모든 자원을 정리하도록 실행 흐름을 분리합니다.
func run(ctx context.Context, cfg config.Config, logger *slog.Logger) (runErr error) {
	db, err := mongodb.Connect(ctx, cfg.MongoDB)
	if err != nil {
		return err
	}
	defer func() { runErr = errors.Join(runErr, db.Close()) }()
	logger.Info("MongoDB connection ready", "environment", cfg.Environment)

	market := domain.Market(cfg.TradingMarket)
	apiClient, err := tossinvest.NewClient(tossinvest.Config{
		ClientID: cfg.ClientID, ClientSecret: cfg.ClientSecret,
		BaseURL: cfg.TossBaseURL, Logger: logger,
	})
	if err != nil {
		return fmt.Errorf("invalid Toss Securities API configuration: %w", err)
	}
	collector, err := marketdata.NewCollector(apiClient, marketdata.Config{})
	if err != nil {
		return fmt.Errorf("create market data collector: %w", err)
	}
	selectedStrategy, err := strategy.NewScoreEngineForMarket(market)
	if err != nil {
		return fmt.Errorf("invalid market strategy: %w", err)
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
		return fmt.Errorf("collect market data: %w", err)
	}
	for _, snapshot := range snapshots {
		if snapshot.IsStale || len(snapshot.MissingFields) > 0 {
			logger.Warn("market snapshot quality warning", "symbol", snapshot.Symbol, "stale", snapshot.IsStale, "missing_fields", snapshot.MissingFields)
		}
		if err := engine.RunOnce(ctx, snapshot); err != nil {
			return fmt.Errorf("trading cycle failed for %s: %w", snapshot.Symbol, err)
		}
	}
	return nil
}
