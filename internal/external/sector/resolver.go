// Package sector는 시장 분류 공급자 어댑터를 둡니다.
package sector

import (
	"context"

	"auto-stock-trading/internal/domain"
)

// PlaceholderResolver는 실제 데이터 공급자가 정해질 때까지 분류 값을 비워 둡니다.
// 빈 값은 marketdata에 의해 MissingFields의 "sector"로 명시됩니다.
type PlaceholderResolver struct{}

func NewPlaceholderResolver() PlaceholderResolver { return PlaceholderResolver{} }

func (PlaceholderResolver) ResolveSector(ctx context.Context, _ domain.Market, _ string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	return "", nil
}
