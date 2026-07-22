// Package domain은 거래 흐름에서 공유하는 핵심 도메인 모델을 정의합니다.
package domain

import "time"

type Side string
type Market string
type Currency string

const (
	SideBuy     Side     = "buy"
	SideSell    Side     = "sell"
	MarketUS    Market   = "us"
	MarketKR    Market   = "kr"
	CurrencyKRW Currency = "KRW"
	CurrencyUSD Currency = "USD"
)

// Candle은 PriceScale이 적용된 OHLC 가격과 정수 거래량을 보관합니다.
type Candle struct {
	Timestamp time.Time
	Open      int64
	High      int64
	Low       int64
	Close     int64
	Volume    int64
}

type MarketSnapshot struct {
	Symbol        string
	Market        Market
	Currency      Currency
	PriceScale    int64
	Price         int64
	Timestamp     time.Time
	CollectedAt   time.Time
	IsStale       bool
	MissingFields []string
	MarketCap     int64
	AverageVolume int64
	IsETF         bool
	Sector        string
	DailyCandles  []Candle
	WeeklyCandles []Candle
	Scores        map[string]float64
}

// Complete는 전략 입력에 필요한 시장 데이터가 모두 수집됐는지 반환합니다.
func (s MarketSnapshot) Complete() bool {
	return !s.IsStale && len(s.MissingFields) == 0
}

type Signal struct {
	Symbol     string
	Side       Side
	Quantity   int64
	Currency   Currency
	PriceScale int64
	Price      int64
}

// Amount는 Price와 같은 최소 통화 단위(KRW 원, USD 센트)의 주문 금액을 반환합니다.
func (s Signal) Amount() int64 {
	return s.Quantity * s.Price
}

type OrderResult struct {
	OrderID string
	Status  string
}
