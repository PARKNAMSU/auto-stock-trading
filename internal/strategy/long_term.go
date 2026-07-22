// 장기 전략의 구체적인 매매 규칙을 구현하기 위한 골격입니다.
package strategy

import (
	"context"

	"auto-stock-trading/internal/domain"
)

// LongTerm represents a long-term trading strategy.
// TODO: 장기 추세, 펀더멘털 등의 진입·청산 규칙을 추가합니다.
type LongTerm struct{}

func NewLongTerm() *LongTerm {
	return &LongTerm{}
}

func (s *LongTerm) Evaluate(context.Context, domain.MarketSnapshot) ([]domain.Signal, error) {
	return nil, nil
}
