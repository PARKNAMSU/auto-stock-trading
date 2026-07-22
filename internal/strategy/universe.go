// 이 파일은 시가총액과 거래량으로 분석 대상 종목을 선별합니다.
package strategy

import (
	"fmt"

	"auto-stock-trading/internal/domain"
)

const (
	DefaultMinMarketCapUS   int64 = 2_000_000_000
	DefaultMinMarketCapKR   int64 = 2_000_000_000_000
	DefaultMinAverageVolume int64 = 1_000_000
)

type UniverseFilter struct {
	Market           domain.Market
	MinMarketCap     int64
	MinAverageVolume int64
}

func NewUniverseFilter() UniverseFilter {
	filter, _ := NewUniverseFilterForMarket(domain.MarketUS)
	return filter
}

func NewUniverseFilterForMarket(market domain.Market) (UniverseFilter, error) {
	var minMarketCap int64
	switch market {
	case domain.MarketUS:
		minMarketCap = DefaultMinMarketCapUS
	case domain.MarketKR:
		minMarketCap = DefaultMinMarketCapKR
	default:
		return UniverseFilter{}, fmt.Errorf("unsupported market %q", market)
	}
	return UniverseFilter{
		Market:           market,
		MinMarketCap:     minMarketCap,
		MinAverageVolume: DefaultMinAverageVolume,
	}, nil
}

func (f UniverseFilter) Includes(snapshot domain.MarketSnapshot) bool {
	if snapshot.Market != f.Market {
		return false
	}
	if snapshot.AverageVolume < f.MinAverageVolume {
		return false
	}
	return snapshot.IsETF || snapshot.MarketCap >= f.MinMarketCap
}
