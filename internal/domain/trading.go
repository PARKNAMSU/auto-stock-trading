// Package domain은 거래 흐름에서 공유하는 핵심 도메인 모델을 정의합니다.
package domain

import "time"

type Side string

const (
	SideBuy  Side = "buy"
	SideSell Side = "sell"
)

type MarketSnapshot struct {
	Symbol    string
	Price     int64
	Timestamp time.Time
}

type Signal struct {
	Symbol   string
	Side     Side
	Quantity int64
	Price    int64
}

func (s Signal) Amount() int64 {
	return s.Quantity * s.Price
}

type OrderResult struct {
	OrderID string
	Status  string
}
