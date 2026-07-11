package trading

import (
	"context"
	"fmt"
	"log/slog"
	"sync/atomic"

	"auto-stock-trading/internal/domain"
)

type DryRunExecutor struct {
	logger *slog.Logger
	nextID atomic.Uint64
}

func NewDryRunExecutor(logger *slog.Logger) *DryRunExecutor {
	return &DryRunExecutor{logger: logger}
}

func (e *DryRunExecutor) Execute(ctx context.Context, signal domain.Signal) (domain.OrderResult, error) {
	if err := ctx.Err(); err != nil {
		return domain.OrderResult{}, err
	}
	id := fmt.Sprintf("dry-%d", e.nextID.Add(1))
	e.logger.Info("dry-run order", "symbol", signal.Symbol, "side", signal.Side, "quantity", signal.Quantity, "price", signal.Price)
	return domain.OrderResult{OrderID: id, Status: "simulated"}, nil
}
