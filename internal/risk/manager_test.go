// 이 파일은 주문 한도에 따른 위험관리자의 승인 및 거절 동작을 검증합니다.
package risk

import (
	"errors"
	"testing"

	"auto-stock-trading/internal/domain"
)

func TestManagerRejectsOrderAboveLimit(t *testing.T) {
	manager := NewManager(100_000)
	signal := domain.Signal{Symbol: "005930", Side: domain.SideBuy, Quantity: 2, Price: 60_000}

	if err := manager.Validate(signal); !errors.Is(err, ErrRejected) {
		t.Fatalf("expected ErrRejected, got %v", err)
	}
}

func TestManagerAcceptsValidOrder(t *testing.T) {
	manager := NewManager(100_000)
	signal := domain.Signal{Symbol: "005930", Side: domain.SideBuy, Quantity: 1, Price: 60_000}

	if err := manager.Validate(signal); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
