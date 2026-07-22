// 이 파일은 토스증권에 실제 주문을 전달할 실행기의 안전한 골격을 제공합니다.
package trading

import (
	"context"
	"errors"

	"auto-stock-trading/internal/domain"
	"auto-stock-trading/internal/tossinvest"
)

// ErrLiveExecutionNotImplemented는 실제 주문 전송 로직이 아직 연결되지 않았음을 나타냅니다.
// 주문 API 구현 전까지 LiveExecutor가 실수로 주문 성공을 가장하지 않도록 명시적으로 반환합니다.
var ErrLiveExecutionNotImplemented = errors.New("live order execution is not implemented")

// LiveExecutor는 토스증권 공통 클라이언트로 실제 주문을 실행할 Executor 구현체입니다.
// 현재는 후속 작업에서 주문 API 모델을 연결할 수 있도록 의존성과 인터페이스만 구성합니다.
type LiveExecutor struct {
	client *tossinvest.Client
}

// NewLiveExecutor는 실제 주문에 사용할 인증된 토스증권 클라이언트를 주입합니다.
func NewLiveExecutor(client *tossinvest.Client) (*LiveExecutor, error) {
	if client == nil {
		return nil, errors.New("tossinvest client is required for live execution")
	}
	return &LiveExecutor{client: client}, nil
}

// Execute는 실제 주문 실행 진입점입니다.
// 현재는 컨텍스트 취소를 먼저 확인한 뒤 미구현 오류를 반환하며 네트워크 요청은 전송하지 않습니다.
func (e *LiveExecutor) Execute(ctx context.Context, _ domain.Signal) (domain.OrderResult, error) {
	if err := ctx.Err(); err != nil {
		return domain.OrderResult{}, err
	}
	return domain.OrderResult{}, ErrLiveExecutionNotImplemented
}

// LiveExecutor가 거래 엔진이 요구하는 Executor 계약을 계속 만족하는지 컴파일 시점에 검증합니다.
var _ Executor = (*LiveExecutor)(nil)
