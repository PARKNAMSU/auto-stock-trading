// Package strategy는 매매 신호를 생성하는 전략 인터페이스와 기본 구현을 제공합니다.
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
