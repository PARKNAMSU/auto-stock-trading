// Package risk는 주문 실행 전에 매매 신호의 위험 조건을 검증합니다.
package risk

import (
	"errors"
	"fmt"

	"auto-stock-trading/internal/domain"
)

var ErrRejected = errors.New("order rejected by risk manager")

type Manager struct {
	maxOrderAmount int64
}

func NewManager(maxOrderAmount int64) *Manager {
	return &Manager{maxOrderAmount: maxOrderAmount}
}

func (m *Manager) Validate(signal domain.Signal) error {
	if signal.Symbol == "" || signal.Quantity <= 0 || signal.Price <= 0 {
		return fmt.Errorf("%w: invalid signal", ErrRejected)
	}
	if signal.Side != domain.SideBuy && signal.Side != domain.SideSell {
		return fmt.Errorf("%w: invalid side", ErrRejected)
	}
	if signal.Amount() > m.maxOrderAmount {
		return fmt.Errorf("%w: amount %d exceeds limit %d", ErrRejected, signal.Amount(), m.maxOrderAmount)
	}
	return nil
}
