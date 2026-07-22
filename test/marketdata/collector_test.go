// 이 파일은 외부 패키지 관점에서 시장 데이터 수집과 스냅샷 정규화를 검증합니다.
package marketdata_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"auto-stock-trading/internal/domain"
	"auto-stock-trading/internal/marketdata"
)

func TestSymbolsUsesMarketRanking(t *testing.T) {
	client := &fakeClient{responses: map[string]string{
		"/api/v1/rankings": `{"result":{"rankedAt":"2026-03-25T09:30:00+09:00","rankings":[{"symbol":"005930"},{"symbol":"000660"},{"symbol":"005930"}]}}`,
	}}
	collector := newCollector(t, client, marketdata.Config{UniverseCount: 3})

	symbols, err := collector.Symbols(context.Background(), domain.MarketKR)
	if err != nil {
		t.Fatalf("Symbols(): %v", err)
	}
	if got, want := strings.Join(symbols, ","), "005930,000660"; got != want {
		t.Fatalf("symbols = %q, want %q", got, want)
	}
	request := client.lastRequest(t)
	if got := request.URL.Query().Get("marketCountry"); got != "KR" {
		t.Fatalf("marketCountry = %q", got)
	}
	if got := request.URL.Query().Get("type"); got != "MARKET_TRADING_AMOUNT" {
		t.Fatalf("type = %q", got)
	}
	if got := request.URL.Query().Get("count"); got != "3" {
		t.Fatalf("count = %q", got)
	}
}

func TestSnapshotBuildsFreshKRMarketData(t *testing.T) {
	now := time.Date(2026, 3, 25, 9, 35, 0, 0, time.FixedZone("KST", 9*60*60))
	client := stockClient("005930", "KRW", "72000", "5919637922", "ETF", "2026-03-25T09:30:00+09:00", []string{
		candle("2026-03-25T09:00:00+09:00", "71600", "72300", "71500", "72000", "300", "KRW"),
		candle("2026-03-24T09:00:00+09:00", "71000", "71800", "70900", "71600", "200", "KRW"),
		candle("2026-03-20T09:00:00+09:00", "70000", "71200", "69900", "71000", "100", "KRW"),
	})
	collector := newCollector(t, client, marketdata.Config{CandleCount: 3, AverageVolumeDays: 3, MaxAge: 10 * time.Minute, Now: func() time.Time { return now }, SectorResolver: fixedSector("Technology")})

	snapshot, err := collector.Snapshot(context.Background(), domain.MarketKR, "005930")
	if err != nil {
		t.Fatalf("Snapshot(): %v", err)
	}
	if snapshot.Currency != domain.CurrencyKRW || snapshot.Price != 72_000 || snapshot.PriceScale != 1 {
		t.Fatalf("unexpected normalized price: %+v", snapshot)
	}
	if snapshot.MarketCap != 426_213_930_384_000 {
		t.Fatalf("MarketCap = %d", snapshot.MarketCap)
	}
	if snapshot.AverageVolume != 200 {
		t.Fatalf("AverageVolume = %d", snapshot.AverageVolume)
	}
	if !snapshot.IsETF || snapshot.Sector != "Technology" || snapshot.IsStale || !snapshot.Complete() {
		t.Fatalf("unexpected quality fields: %+v", snapshot)
	}
	if len(snapshot.DailyCandles) != 3 || len(snapshot.WeeklyCandles) != 2 {
		t.Fatalf("daily=%d weekly=%d", len(snapshot.DailyCandles), len(snapshot.WeeklyCandles))
	}
	latestWeek := snapshot.WeeklyCandles[0]
	if latestWeek.Open != 71_000 || latestWeek.Close != 72_000 || latestWeek.High != 72_300 || latestWeek.Low != 70_900 || latestWeek.Volume != 500 {
		t.Fatalf("unexpected weekly candle: %+v", latestWeek)
	}
}

func TestSnapshotNormalizesUSDToCents(t *testing.T) {
	now := time.Date(2026, 3, 25, 22, 31, 0, 0, time.FixedZone("KST", 9*60*60))
	client := stockClient("AAPL", "USD", "185.70", "14702703000", "STOCK", "2026-03-25T22:30:00+09:00", []string{
		candle("2026-03-25T09:00:00+09:00", "184.10", "186.25", "183.90", "185.70", "1000", "USD"),
	})
	collector := newCollector(t, client, marketdata.Config{CandleCount: 1, AverageVolumeDays: 1, Now: func() time.Time { return now }, SectorResolver: fixedSector("Technology")})

	snapshot, err := collector.Snapshot(context.Background(), domain.MarketUS, "AAPL")
	if err != nil {
		t.Fatalf("Snapshot(): %v", err)
	}
	if snapshot.Price != 18_570 || snapshot.PriceScale != 100 {
		t.Fatalf("price=%d scale=%d", snapshot.Price, snapshot.PriceScale)
	}
	if snapshot.MarketCap != 2_730_291_947_100 {
		t.Fatalf("MarketCap = %d", snapshot.MarketCap)
	}
	if snapshot.DailyCandles[0].High != 18_625 {
		t.Fatalf("high = %d", snapshot.DailyCandles[0].High)
	}
}

func TestSnapshotIdentifiesStaleAndIncompleteData(t *testing.T) {
	now := time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)
	client := stockClient("AAPL", "USD", "185.70", "100", "STOCK", "", nil)
	collector := newCollector(t, client, marketdata.Config{CandleCount: 1, AverageVolumeDays: 1, MaxAge: time.Minute, Now: func() time.Time { return now }})

	snapshot, err := collector.Snapshot(context.Background(), domain.MarketUS, "AAPL")
	if err != nil {
		t.Fatalf("Snapshot(): %v", err)
	}
	if !snapshot.IsStale || snapshot.Complete() {
		t.Fatalf("snapshot quality = stale:%v missing:%v", snapshot.IsStale, snapshot.MissingFields)
	}
	if got, want := strings.Join(snapshot.MissingFields, ","), "timestamp,dailyCandles,sector"; got != want {
		t.Fatalf("missing fields = %q, want %q", got, want)
	}
}

func TestSnapshotReturnsAPIError(t *testing.T) {
	want := errors.New("prices unavailable")
	client := &fakeClient{errors: map[string]error{"/api/v1/prices": want}}
	collector := newCollector(t, client, marketdata.Config{})

	_, err := collector.Snapshot(context.Background(), domain.MarketKR, "005930")
	if !errors.Is(err, want) {
		t.Fatalf("Snapshot() error = %v, want wrapped API error", err)
	}
}

type fixedSector string

func (s fixedSector) ResolveSector(context.Context, domain.Market, string) (string, error) {
	return string(s), nil
}

type fakeClient struct {
	responses map[string]string
	errors    map[string]error
	mu        sync.Mutex
	requests  []*http.Request
}

func (f *fakeClient) NewRequest(ctx context.Context, method, path string, body io.Reader) (*http.Request, error) {
	return http.NewRequestWithContext(ctx, method, "https://example.test"+path, body)
}

func (f *fakeClient) Do(req *http.Request) (*http.Response, error) {
	f.mu.Lock()
	f.requests = append(f.requests, req)
	f.mu.Unlock()
	if err := f.errors[req.URL.Path]; err != nil {
		return nil, err
	}
	payload, ok := f.responses[req.URL.Path]
	if !ok {
		return nil, errors.New("unexpected request: " + req.URL.Path)
	}
	return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(payload)), Request: req}, nil
}

func (f *fakeClient) lastRequest(t *testing.T) *http.Request {
	t.Helper()
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.requests) == 0 {
		t.Fatal("no requests recorded")
	}
	return f.requests[len(f.requests)-1]
}

func newCollector(t *testing.T, client marketdata.Client, cfg marketdata.Config) *marketdata.Collector {
	t.Helper()
	collector, err := marketdata.NewCollector(client, cfg)
	if err != nil {
		t.Fatalf("NewCollector(): %v", err)
	}
	return collector
}

func stockClient(symbol, currency, price, shares, securityType, timestamp string, candles []string) *fakeClient {
	timestampJSON := "null"
	if timestamp != "" {
		timestampJSON = `"` + timestamp + `"`
	}
	return &fakeClient{responses: map[string]string{
		"/api/v1/prices":  `{"result":[{"symbol":"` + symbol + `","timestamp":` + timestampJSON + `,"lastPrice":"` + price + `","currency":"` + currency + `"}]}`,
		"/api/v1/stocks":  `{"result":[{"symbol":"` + symbol + `","securityType":"` + securityType + `","currency":"` + currency + `","sharesOutstanding":"` + shares + `"}]}`,
		"/api/v1/candles": `{"result":{"candles":[` + strings.Join(candles, ",") + `]}}`,
	}}
}

func candle(timestamp, open, high, low, closePrice, volume, currency string) string {
	return `{"timestamp":"` + timestamp + `","openPrice":"` + open + `","highPrice":"` + high + `","lowPrice":"` + low + `","closePrice":"` + closePrice + `","volume":"` + volume + `","currency":"` + currency + `"}`
}
