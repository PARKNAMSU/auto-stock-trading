// 이 파일은 외부 패키지 관점에서 실제 주문 실행기 골격의 안전한 기본 동작을 검증합니다.
package trading_test

import (
	"context"
	"errors"
	"testing"

	"auto-stock-trading/internal/domain"
	"auto-stock-trading/internal/tossinvest"
	"auto-stock-trading/internal/trading"
)

func TestNewLiveExecutorRequiresClient(t *testing.T) {
	if _, err := trading.NewLiveExecutor(nil); err == nil {
		t.Fatal("NewLiveExecutor() accepted a nil client")
	}
}

func TestLiveExecutorReturnsNotImplementedWithoutSendingOrder(t *testing.T) {
	executor := newLiveExecutor(t)

	result, err := executor.Execute(context.Background(), domain.Signal{
		Symbol: "005930", Side: domain.SideBuy, Quantity: 1, Price: 70_000,
	})
	if !errors.Is(err, trading.ErrLiveExecutionNotImplemented) {
		t.Fatalf("Execute() error = %v, want ErrLiveExecutionNotImplemented", err)
	}
	if result != (domain.OrderResult{}) {
		t.Fatalf("Execute() result = %+v, want zero value", result)
	}
}

func TestLiveExecutorHonorsCanceledContext(t *testing.T) {
	executor := newLiveExecutor(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := executor.Execute(ctx, domain.Signal{})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Execute() error = %v, want context.Canceled", err)
	}
}

func newLiveExecutor(t *testing.T) *trading.LiveExecutor {
	t.Helper()
	client, err := tossinvest.NewClient(tossinvest.Config{
		ClientID: "test-client", ClientSecret: "test-secret",
	})
	if err != nil {
		t.Fatalf("tossinvest.NewClient(): %v", err)
	}
	executor, err := trading.NewLiveExecutor(client)
	if err != nil {
		t.Fatalf("NewLiveExecutor(): %v", err)
	}
	return executor
}
