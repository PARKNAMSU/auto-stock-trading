package sector_test

import (
	"context"
	"errors"
	"testing"

	"auto-stock-trading/internal/domain"
	"auto-stock-trading/internal/external/sector"
)

func TestPlaceholderResolverReturnsNoUnverifiedClassification(t *testing.T) {
	got, err := sector.NewPlaceholderResolver().ResolveSector(context.Background(), domain.MarketUS, "AAPL")
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Fatalf("placeholder returned an unverified sector %q", got)
	}
}

func TestPlaceholderResolverHonorsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	got, err := sector.NewPlaceholderResolver().ResolveSector(ctx, domain.MarketKR, "005930")
	if got != "" || !errors.Is(err, context.Canceled) {
		t.Fatalf("expected canceled context and empty result, got %q, %v", got, err)
	}
}
