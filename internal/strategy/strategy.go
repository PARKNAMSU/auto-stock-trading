// Package strategy는 매매 신호를 생성하는 전략 인터페이스와 기본 구현을 제공합니다.
package strategy

import (
	"context"
	"fmt"

	"auto-stock-trading/internal/domain"
)

type Strategy interface {
	Evaluate(context.Context, domain.MarketSnapshot) ([]domain.Signal, error)
}

// Kind identifies a strategy that can be selected at runtime.
type Kind string

const (
	KindHold       Kind = "hold"
	KindShortTerm  Kind = "short-term"
	KindMediumTerm Kind = "medium-term"
	KindLongTerm   Kind = "long-term"
)

// New creates the strategy selected by kind.
func New(kind Kind) (Strategy, error) {
	switch kind {
	case KindHold:
		return NewHold(), nil
	case KindShortTerm:
		return NewShortTerm(), nil
	case KindMediumTerm:
		return NewMediumTerm(), nil
	case KindLongTerm:
		return NewLongTerm(), nil
	default:
		return nil, fmt.Errorf("unsupported strategy %q", kind)
	}
}

// Hold is a safe placeholder strategy that never creates an order signal.
type Hold struct{}

func NewHold() Hold {
	return Hold{}
}

func (Hold) Evaluate(context.Context, domain.MarketSnapshot) ([]domain.Signal, error) {
	return nil, nil
}
