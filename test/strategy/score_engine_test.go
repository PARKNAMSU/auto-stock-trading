// 이 파일은 외부 패키지 관점에서 유니버스 필터와 종합 점수에 따른 신호 생성을 검증합니다.
package strategy_test

import (
	"context"
	"testing"

	"auto-stock-trading/internal/domain"
	"auto-stock-trading/internal/strategy"
)

func TestScoreEngineCreatesBuySignalAboveThreshold(t *testing.T) {
	engine := strategy.NewScoreEngine()
	snapshot := eligibleSnapshot()
	snapshot.Scores = highScores()

	signals, err := engine.Evaluate(context.Background(), snapshot)
	if err != nil {
		t.Fatalf("Evaluate() returned an error: %v", err)
	}
	if len(signals) != 1 || signals[0].Side != domain.SideBuy {
		t.Fatalf("expected one buy signal, got %#v", signals)
	}
}

func TestScoreEngineRejectsLowScore(t *testing.T) {
	engine := strategy.NewScoreEngine()
	snapshot := eligibleSnapshot()
	snapshot.Scores = map[string]float64{strategy.ScoreMomentum: 50}

	signals, err := engine.Evaluate(context.Background(), snapshot)
	if err != nil {
		t.Fatalf("Evaluate() returned an error: %v", err)
	}
	if len(signals) != 0 {
		t.Fatalf("expected no signals, got %#v", signals)
	}
}

func TestScoreEngineAppliesUniverseFilter(t *testing.T) {
	engine := strategy.NewScoreEngine()
	snapshot := eligibleSnapshot()
	snapshot.AverageVolume = 999_999

	signals, err := engine.Evaluate(context.Background(), snapshot)
	if err != nil {
		t.Fatalf("Evaluate() returned an error: %v", err)
	}
	if len(signals) != 0 {
		t.Fatalf("expected no signals, got %#v", signals)
	}
}

func TestScoreEngineUsesKRUniverse(t *testing.T) {
	engine, err := strategy.NewScoreEngineForMarket(domain.MarketKR)
	if err != nil {
		t.Fatalf("NewScoreEngineForMarket() returned an error: %v", err)
	}
	snapshot := eligibleSnapshot()
	snapshot.Market = domain.MarketKR
	snapshot.MarketCap = strategy.DefaultMinMarketCapKR
	snapshot.Scores = highScores()

	signals, err := engine.Evaluate(context.Background(), snapshot)
	if err != nil {
		t.Fatalf("Evaluate() returned an error: %v", err)
	}
	if len(signals) != 1 {
		t.Fatalf("expected one signal, got %#v", signals)
	}
}

func TestScoreEngineRejectsSnapshotFromDifferentMarket(t *testing.T) {
	engine := strategy.NewScoreEngine()
	snapshot := eligibleSnapshot()
	snapshot.Market = domain.MarketKR
	snapshot.MarketCap = strategy.DefaultMinMarketCapKR
	snapshot.Scores = highScores()

	signals, err := engine.Evaluate(context.Background(), snapshot)
	if err != nil {
		t.Fatalf("Evaluate() returned an error: %v", err)
	}
	if len(signals) != 0 {
		t.Fatalf("expected no signals, got %#v", signals)
	}
}

func TestScoreEngineRejectsInvalidComponentScore(t *testing.T) {
	engine := strategy.NewScoreEngine()
	snapshot := eligibleSnapshot()
	snapshot.Scores = map[string]float64{strategy.ScoreMomentum: 101}

	if _, err := engine.Evaluate(context.Background(), snapshot); err == nil {
		t.Fatal("Evaluate() accepted an invalid score")
	}
}

func TestNewScoreEngineWithConfigRejectsInvalidThreshold(t *testing.T) {
	config := strategy.DefaultScoreEngineConfig()
	config.BuyThreshold = 101

	if _, err := strategy.NewScoreEngineWithConfig(config); err == nil {
		t.Fatal("NewScoreEngineWithConfig() accepted an invalid threshold")
	}
}

func eligibleSnapshot() domain.MarketSnapshot {
	return domain.MarketSnapshot{
		Symbol:        "NVDA",
		Market:        domain.MarketUS,
		Price:         150,
		MarketCap:     strategy.DefaultMinMarketCapUS,
		AverageVolume: strategy.DefaultMinAverageVolume,
	}
}

func highScores() map[string]float64 {
	return map[string]float64{
		strategy.ScoreRelativeStrength: 90,
		strategy.ScoreMomentum:         90,
		strategy.ScoreTrendFollowing:   90,
		strategy.ScoreSectorRotation:   90,
		strategy.ScoreEvent:            90,
	}
}

var _ strategy.Strategy = (*strategy.ScoreEngine)(nil)
