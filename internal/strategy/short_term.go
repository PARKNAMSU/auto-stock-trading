// 단기 전략의 구체적인 매매 규칙을 구현하기 위한 골격입니다.
package strategy

import (
	"context"

	"auto-stock-trading/internal/domain"
)

// ShortTerm represents a short-term trading strategy.
// TODO: 단기 가격 변동, 거래량 등의 진입·청산 규칙을 추가합니다.
type ShortTerm struct{}

func NewShortTerm() *ShortTerm {
	return &ShortTerm{}
}

func (s *ShortTerm) Evaluate(context.Context, domain.MarketSnapshot) ([]domain.Signal, error) {
	return nil, nil
}
