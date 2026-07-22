// 중기 전략의 구체적인 매매 규칙을 구현하기 위한 골격입니다.
package strategy

import (
	"context"

	"auto-stock-trading/internal/domain"
)

// MediumTerm represents a medium-term trading strategy.
// TODO: 추세, 이동평균 등의 진입·청산 규칙을 추가합니다.
type MediumTerm struct{}

func NewMediumTerm() *MediumTerm {
	return &MediumTerm{}
}

func (s *MediumTerm) Evaluate(context.Context, domain.MarketSnapshot) ([]domain.Signal, error) {
	return nil, nil
}
