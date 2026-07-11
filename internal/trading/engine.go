package trading

import (
	"context"
	"fmt"
	"log/slog"

	"auto-stock-trading/internal/domain"
	"auto-stock-trading/internal/strategy"
)

type RiskManager interface {
	Validate(domain.Signal) error
}

type Executor interface {
	Execute(context.Context, domain.Signal) (domain.OrderResult, error)
}

type Engine struct {
	strategy strategy.Strategy
	risk     RiskManager
	executor Executor
	logger   *slog.Logger
}

func NewEngine(strategy strategy.Strategy, risk RiskManager, executor Executor, logger *slog.Logger) *Engine {
	return &Engine{strategy: strategy, risk: risk, executor: executor, logger: logger}
}

func (e *Engine) RunOnce(ctx context.Context, snapshot domain.MarketSnapshot) error {
	signals, err := e.strategy.Evaluate(ctx, snapshot)
	if err != nil {
		return fmt.Errorf("evaluate strategy: %w", err)
	}

	for _, signal := range signals {
		if err := e.risk.Validate(signal); err != nil {
			e.logger.Warn("signal rejected", "symbol", signal.Symbol, "error", err)
			continue
		}
		result, err := e.executor.Execute(ctx, signal)
		if err != nil {
			return fmt.Errorf("execute order for %s: %w", signal.Symbol, err)
		}
		e.logger.Info("order processed", "symbol", signal.Symbol, "order_id", result.OrderID, "status", result.Status)
	}
	return nil
}
