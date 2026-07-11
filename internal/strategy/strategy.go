package strategy

import (
	"context"

	"auto-stock-trading/internal/domain"
)

type Strategy interface {
	Evaluate(context.Context, domain.MarketSnapshot) ([]domain.Signal, error)
}

// Hold is a safe placeholder strategy that never creates an order signal.
type Hold struct{}

func NewHold() Hold {
	return Hold{}
}

func (Hold) Evaluate(context.Context, domain.MarketSnapshot) ([]domain.Signal, error) {
	return nil, nil
}
