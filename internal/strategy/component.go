// 이 파일은 전략 엔진에서 조합할 개별 점수 전략을 정의합니다.
package strategy

import (
	"context"
	"fmt"

	"auto-stock-trading/internal/domain"
)

const (
	ScoreRelativeStrength = "relative-strength"
	ScoreMomentum         = "momentum"
	ScoreTrendFollowing   = "trend-following"
	ScoreMeanReversion    = "mean-reversion"
	ScoreBreakout         = "breakout"
	ScoreSectorRotation   = "sector-rotation"
	ScoreEarnings         = "earnings"
	ScoreEvent            = "event"
	ScoreAINews           = "ai-news"
)

// Component gives one aspect of a market snapshot a score from 0 to 100.
type Component interface {
	Name() string
	Score(context.Context, domain.MarketSnapshot) (float64, error)
}

type scoredComponent struct {
	name string
}

func (s scoredComponent) Name() string {
	return s.name
}

func (s scoredComponent) Score(ctx context.Context, snapshot domain.MarketSnapshot) (float64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	score, ok := snapshot.Scores[s.name]
	if !ok {
		return 0, nil
	}
	if score < 0 || score > 100 {
		return 0, fmt.Errorf("%s score must be between 0 and 100", s.name)
	}
	return score, nil
}

func NewRelativeStrength() Component { return scoredComponent{name: ScoreRelativeStrength} }
func NewMomentum() Component         { return scoredComponent{name: ScoreMomentum} }
func NewTrendFollowing() Component   { return scoredComponent{name: ScoreTrendFollowing} }
func NewMeanReversion() Component    { return scoredComponent{name: ScoreMeanReversion} }
func NewBreakout() Component         { return scoredComponent{name: ScoreBreakout} }
func NewSectorRotation() Component   { return scoredComponent{name: ScoreSectorRotation} }
func NewEarnings() Component         { return scoredComponent{name: ScoreEarnings} }
func NewEventFilter() Component      { return scoredComponent{name: ScoreEvent} }
func NewAINews() Component           { return scoredComponent{name: ScoreAINews} }
