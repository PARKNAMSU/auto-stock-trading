// 이 파일은 여러 전략 점수를 가중 평균해 최종 매매 신호를 생성합니다.
package strategy

import (
	"context"
	"fmt"

	"auto-stock-trading/internal/domain"
)

const DefaultBuyThreshold = 85.0

type WeightedComponent struct {
	Component Component
	Weight    float64
}

// ScoreEngineConfig allows components, weights, filters, and the buy threshold
// to be changed without modifying the trading engine.
type ScoreEngineConfig struct {
	Universe     UniverseFilter
	Components   []WeightedComponent
	BuyThreshold float64
	Quantity     int64
}

type ScoreEngine struct {
	universe     UniverseFilter
	components   []WeightedComponent
	buyThreshold float64
	quantity     int64
}

// NewScoreEngine builds the PDF-recommended relative-strength, momentum,
// trend-following, sector-rotation, and event-filter strategy combination.
func NewScoreEngine() *ScoreEngine {
	engine, err := NewScoreEngineForMarket(domain.MarketUS)
	if err != nil {
		panic(err)
	}
	return engine
}

func DefaultScoreEngineConfig() ScoreEngineConfig {
	config, _ := DefaultScoreEngineConfigForMarket(domain.MarketUS)
	return config
}

func NewScoreEngineForMarket(market domain.Market) (*ScoreEngine, error) {
	config, err := DefaultScoreEngineConfigForMarket(market)
	if err != nil {
		return nil, err
	}
	return NewScoreEngineWithConfig(config)
}

func DefaultScoreEngineConfigForMarket(market domain.Market) (ScoreEngineConfig, error) {
	universe, err := NewUniverseFilterForMarket(market)
	if err != nil {
		return ScoreEngineConfig{}, err
	}
	return ScoreEngineConfig{
		Universe: universe,
		Components: []WeightedComponent{
			{Component: NewRelativeStrength(), Weight: 1},
			{Component: NewMomentum(), Weight: 1},
			{Component: NewTrendFollowing(), Weight: 1},
			{Component: NewSectorRotation(), Weight: 1},
			{Component: NewEventFilter(), Weight: 1},
		},
		BuyThreshold: DefaultBuyThreshold,
		Quantity:     1,
	}, nil
}

func NewScoreEngineWithConfig(config ScoreEngineConfig) (*ScoreEngine, error) {
	if config.BuyThreshold < 0 || config.BuyThreshold > 100 {
		return nil, fmt.Errorf("buy threshold must be between 0 and 100")
	}
	if config.Quantity <= 0 {
		return nil, fmt.Errorf("quantity must be positive")
	}
	if len(config.Components) == 0 {
		return nil, fmt.Errorf("at least one strategy component is required")
	}
	components := append([]WeightedComponent(nil), config.Components...)
	return &ScoreEngine{
		universe:     config.Universe,
		components:   components,
		buyThreshold: config.BuyThreshold,
		quantity:     config.Quantity,
	}, nil
}

func (e *ScoreEngine) Evaluate(ctx context.Context, snapshot domain.MarketSnapshot) ([]domain.Signal, error) {
	if !e.universe.Includes(snapshot) {
		return nil, nil
	}

	var weightedTotal, totalWeight float64
	for _, candidate := range e.components {
		if candidate.Component == nil || candidate.Weight <= 0 {
			return nil, fmt.Errorf("strategy component and weight must be valid")
		}
		score, err := candidate.Component.Score(ctx, snapshot)
		if err != nil {
			return nil, fmt.Errorf("score %s: %w", candidate.Component.Name(), err)
		}
		weightedTotal += score * candidate.Weight
		totalWeight += candidate.Weight
	}

	if totalWeight == 0 || weightedTotal/totalWeight < e.buyThreshold {
		return nil, nil
	}
	return []domain.Signal{{
		Symbol:     snapshot.Symbol,
		Side:       domain.SideBuy,
		Quantity:   e.quantity,
		Currency:   snapshot.Currency,
		PriceScale: snapshot.PriceScale,
		Price:      snapshot.Price,
	}}, nil
}
