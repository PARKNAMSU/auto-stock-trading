// Package marketdata는 토스증권 시장 데이터를 전략용 스냅샷으로 변환합니다.
package marketdata

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"auto-stock-trading/internal/domain"
)

const (
	defaultUniverseCount = 100
	defaultCandleCount   = 100
	defaultAverageDays   = 20
	defaultMaxAge        = 15 * time.Minute
	maxBatchSize         = 200
)

// Client는 수집기가 사용하는 인증 HTTP 클라이언트 계약입니다.
type Client interface {
	NewRequest(context.Context, string, string, io.Reader) (*http.Request, error)
	Do(*http.Request) (*http.Response, error)
}

// SectorResolver는 공식 API에 없는 섹터 정보를 외부 데이터 소스로 보완합니다.
type SectorResolver interface {
	ResolveSector(context.Context, domain.Market, string) (string, error)
}

// Config는 수집 대상 수, 가격 이력 길이와 데이터 신선도 기준을 정의합니다.
type Config struct {
	UniverseCount     int
	CandleCount       int
	AverageVolumeDays int
	MaxAge            time.Duration
	Now               func() time.Time
	SectorResolver    SectorResolver
}

// Collector는 토스증권의 랭킹·종목·현재가·캔들 응답을 MarketSnapshot으로 조합합니다.
type Collector struct {
	client            Client
	universeCount     int
	candleCount       int
	averageVolumeDays int
	maxAge            time.Duration
	now               func() time.Time
	sectorResolver    SectorResolver
}

// NewCollector는 설정을 검증하고 시장 데이터 수집기를 생성합니다.
func NewCollector(client Client, cfg Config) (*Collector, error) {
	if client == nil {
		return nil, errors.New("market data client is required")
	}
	if cfg.UniverseCount == 0 {
		cfg.UniverseCount = defaultUniverseCount
	}
	if cfg.CandleCount == 0 {
		cfg.CandleCount = defaultCandleCount
	}
	if cfg.AverageVolumeDays == 0 {
		cfg.AverageVolumeDays = defaultAverageDays
		if cfg.CandleCount < cfg.AverageVolumeDays {
			cfg.AverageVolumeDays = cfg.CandleCount
		}
	}
	if cfg.MaxAge == 0 {
		cfg.MaxAge = defaultMaxAge
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	if cfg.UniverseCount < 1 || cfg.UniverseCount > 100 {
		return nil, errors.New("universe count must be between 1 and 100")
	}
	if cfg.CandleCount < 1 || cfg.CandleCount > maxBatchSize {
		return nil, errors.New("candle count must be between 1 and 200")
	}
	if cfg.AverageVolumeDays < 1 || cfg.AverageVolumeDays > cfg.CandleCount {
		return nil, errors.New("average volume days must be between 1 and candle count")
	}
	if cfg.MaxAge < 0 {
		return nil, errors.New("max age cannot be negative")
	}
	return &Collector{client: client, universeCount: cfg.UniverseCount, candleCount: cfg.CandleCount, averageVolumeDays: cfg.AverageVolumeDays, maxAge: cfg.MaxAge, now: cfg.Now, sectorResolver: cfg.SectorResolver}, nil
}

// Symbols는 시장 거래대금 상위 랭킹에서 거래 대상 심볼을 조회합니다.
func (c *Collector) Symbols(ctx context.Context, market domain.Market) ([]string, error) {
	country, err := marketCountry(market)
	if err != nil {
		return nil, err
	}
	query := url.Values{"type": {"MARKET_TRADING_AMOUNT"}, "marketCountry": {country}, "duration": {"realtime"}, "excludeInvestmentCaution": {"true"}, "count": {strconv.Itoa(c.universeCount)}}
	var envelope struct {
		Result struct {
			RankedAt *time.Time `json:"rankedAt"`
			Rankings []struct {
				Symbol string `json:"symbol"`
			} `json:"rankings"`
		} `json:"result"`
	}
	if err := c.getJSON(ctx, "/api/v1/rankings?"+query.Encode(), &envelope); err != nil {
		return nil, fmt.Errorf("get market rankings: %w", err)
	}
	symbols := make([]string, 0, len(envelope.Result.Rankings))
	seen := make(map[string]struct{}, len(envelope.Result.Rankings))
	for _, item := range envelope.Result.Rankings {
		symbol := strings.TrimSpace(item.Symbol)
		if symbol == "" {
			continue
		}
		if _, exists := seen[symbol]; exists {
			continue
		}
		seen[symbol] = struct{}{}
		symbols = append(symbols, symbol)
	}
	if len(symbols) == 0 {
		return nil, errors.New("market rankings returned no symbols")
	}
	return symbols, nil
}

// Snapshot은 한 종목의 현재가, 종목 정보와 수정 일봉을 수집해 정규화합니다.
func (c *Collector) Snapshot(ctx context.Context, market domain.Market, symbol string) (domain.MarketSnapshot, error) {
	if _, err := marketCountry(market); err != nil {
		return domain.MarketSnapshot{}, err
	}
	symbol = strings.TrimSpace(symbol)
	if symbol == "" {
		return domain.MarketSnapshot{}, errors.New("symbol is required")
	}

	price, err := c.price(ctx, symbol)
	if err != nil {
		return domain.MarketSnapshot{}, fmt.Errorf("collect price for %s: %w", symbol, err)
	}
	stock, err := c.stock(ctx, symbol)
	if err != nil {
		return domain.MarketSnapshot{}, fmt.Errorf("collect stock info for %s: %w", symbol, err)
	}
	candles, err := c.candles(ctx, symbol, price.Currency)
	if err != nil {
		return domain.MarketSnapshot{}, fmt.Errorf("collect candles for %s: %w", symbol, err)
	}
	if price.Symbol != symbol || stock.Symbol != symbol {
		return domain.MarketSnapshot{}, errors.New("market data response symbol mismatch")
	}
	if price.Currency != stock.Currency {
		return domain.MarketSnapshot{}, errors.New("price and stock currencies do not match")
	}
	if err := validateMarketCurrency(market, price.Currency); err != nil {
		return domain.MarketSnapshot{}, err
	}

	collectedAt := c.now()
	missing := make([]string, 0, 3)
	if price.Timestamp == nil {
		missing = append(missing, "timestamp")
	}
	if len(candles) == 0 {
		missing = append(missing, "dailyCandles")
	}
	sector := ""
	if c.sectorResolver != nil {
		sector, err = c.sectorResolver.ResolveSector(ctx, market, symbol)
		if err != nil {
			return domain.MarketSnapshot{}, fmt.Errorf("resolve sector for %s: %w", symbol, err)
		}
	}
	if strings.TrimSpace(sector) == "" {
		// 토스증권 OpenAPI 1.2.4에는 섹터 필드가 없으므로 결측을 숨기지 않습니다.
		missing = append(missing, "sector")
	}
	priceScale := currencyScale(price.Currency)
	lastPrice, err := parsePrice(price.LastPrice, priceScale)
	if err != nil {
		return domain.MarketSnapshot{}, fmt.Errorf("parse last price: %w", err)
	}
	shares, err := parseInteger(stock.SharesOutstanding)
	if err != nil {
		return domain.MarketSnapshot{}, fmt.Errorf("parse shares outstanding: %w", err)
	}
	marketCap, err := multiplyMajor(lastPrice, shares, priceScale)
	if err != nil {
		return domain.MarketSnapshot{}, fmt.Errorf("calculate market cap: %w", err)
	}
	if lastPrice == 0 {
		missing = append(missing, "price")
	}
	if shares == 0 {
		missing = append(missing, "sharesOutstanding")
	}
	if stock.SecurityType == "" {
		missing = append(missing, "securityType")
	}
	timestamp := time.Time{}
	if price.Timestamp != nil {
		timestamp = *price.Timestamp
	}
	return domain.MarketSnapshot{
		Symbol: symbol, Market: market, Currency: domain.Currency(price.Currency), PriceScale: priceScale,
		Price: lastPrice, Timestamp: timestamp, CollectedAt: collectedAt,
		IsStale:       timestamp.IsZero() || collectedAt.Sub(timestamp) > c.maxAge,
		MissingFields: missing, MarketCap: marketCap, AverageVolume: averageVolume(candles, c.averageVolumeDays),
		IsETF: stock.SecurityType == "ETF" || stock.SecurityType == "FOREIGN_ETF", Sector: sector,
		DailyCandles: candles, WeeklyCandles: aggregateWeekly(candles), Scores: make(map[string]float64),
	}, nil
}

// Snapshots는 랭킹 종목을 차례로 수집하며 종목 하나의 실패도 시장 전체 실패로 반환합니다.
func (c *Collector) Snapshots(ctx context.Context, market domain.Market) ([]domain.MarketSnapshot, error) {
	symbols, err := c.Symbols(ctx, market)
	if err != nil {
		return nil, err
	}
	snapshots := make([]domain.MarketSnapshot, 0, len(symbols))
	for _, symbol := range symbols {
		snapshot, err := c.Snapshot(ctx, market, symbol)
		if err != nil {
			return nil, err
		}
		snapshots = append(snapshots, snapshot)
	}
	return snapshots, nil
}

type priceResponse struct {
	Symbol    string     `json:"symbol"`
	Timestamp *time.Time `json:"timestamp"`
	LastPrice string     `json:"lastPrice"`
	Currency  string     `json:"currency"`
}
type stockResponse struct {
	Symbol            string `json:"symbol"`
	SecurityType      string `json:"securityType"`
	Currency          string `json:"currency"`
	SharesOutstanding string `json:"sharesOutstanding"`
}
type candleResponse struct {
	Timestamp time.Time `json:"timestamp"`
	Open      string    `json:"openPrice"`
	High      string    `json:"highPrice"`
	Low       string    `json:"lowPrice"`
	Close     string    `json:"closePrice"`
	Volume    string    `json:"volume"`
	Currency  string    `json:"currency"`
}

func (c *Collector) price(ctx context.Context, symbol string) (priceResponse, error) {
	var envelope struct {
		Result []priceResponse `json:"result"`
	}
	if err := c.getJSON(ctx, "/api/v1/prices?"+url.Values{"symbols": {symbol}}.Encode(), &envelope); err != nil {
		return priceResponse{}, err
	}
	if len(envelope.Result) != 1 {
		return priceResponse{}, fmt.Errorf("expected one price, got %d", len(envelope.Result))
	}
	return envelope.Result[0], nil
}

func (c *Collector) stock(ctx context.Context, symbol string) (stockResponse, error) {
	var envelope struct {
		Result []stockResponse `json:"result"`
	}
	if err := c.getJSON(ctx, "/api/v1/stocks?"+url.Values{"symbols": {symbol}}.Encode(), &envelope); err != nil {
		return stockResponse{}, err
	}
	if len(envelope.Result) != 1 {
		return stockResponse{}, fmt.Errorf("expected one stock, got %d", len(envelope.Result))
	}
	return envelope.Result[0], nil
}

func (c *Collector) candles(ctx context.Context, symbol, currency string) ([]domain.Candle, error) {
	query := url.Values{"symbol": {symbol}, "interval": {"1d"}, "count": {strconv.Itoa(c.candleCount)}, "adjusted": {"true"}}
	var envelope struct {
		Result struct {
			Candles []candleResponse `json:"candles"`
		} `json:"result"`
	}
	if err := c.getJSON(ctx, "/api/v1/candles?"+query.Encode(), &envelope); err != nil {
		return nil, err
	}
	scale := currencyScale(currency)
	result := make([]domain.Candle, 0, len(envelope.Result.Candles))
	for _, raw := range envelope.Result.Candles {
		if raw.Currency != currency {
			return nil, errors.New("candle currency does not match current price")
		}
		open, err := parsePrice(raw.Open, scale)
		if err != nil {
			return nil, err
		}
		high, err := parsePrice(raw.High, scale)
		if err != nil {
			return nil, err
		}
		low, err := parsePrice(raw.Low, scale)
		if err != nil {
			return nil, err
		}
		closePrice, err := parsePrice(raw.Close, scale)
		if err != nil {
			return nil, err
		}
		volume, err := parseInteger(raw.Volume)
		if err != nil {
			return nil, err
		}
		if raw.Timestamp.IsZero() || low > high || open < low || open > high || closePrice < low || closePrice > high {
			return nil, errors.New("invalid candle data")
		}
		result = append(result, domain.Candle{Timestamp: raw.Timestamp, Open: open, High: high, Low: low, Close: closePrice, Volume: volume})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Timestamp.After(result[j].Timestamp) })
	return result, nil
}

func (c *Collector) getJSON(ctx context.Context, path string, target any) error {
	req, err := c.client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if err := json.NewDecoder(io.LimitReader(resp.Body, 8<<20)).Decode(target); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

func marketCountry(market domain.Market) (string, error) {
	switch market {
	case domain.MarketKR:
		return "KR", nil
	case domain.MarketUS:
		return "US", nil
	default:
		return "", fmt.Errorf("unsupported market %q", market)
	}
}

func validateMarketCurrency(market domain.Market, currency string) error {
	if (market == domain.MarketKR && currency != string(domain.CurrencyKRW)) || (market == domain.MarketUS && currency != string(domain.CurrencyUSD)) {
		return fmt.Errorf("currency %q does not match market %q", currency, market)
	}
	return nil
}

func currencyScale(currency string) int64 {
	if currency == string(domain.CurrencyUSD) {
		return 100
	}
	return 1
}

func parsePrice(value string, scale int64) (int64, error) {
	parts := strings.Split(value, ".")
	if len(parts) > 2 || parts[0] == "" {
		return 0, fmt.Errorf("invalid decimal %q", value)
	}
	whole, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || whole < 0 {
		return 0, fmt.Errorf("invalid decimal %q", value)
	}
	fractionDigits := 0
	for s := scale; s > 1; s /= 10 {
		fractionDigits++
	}
	fraction := ""
	if len(parts) == 2 {
		fraction = parts[1]
	}
	if len(fraction) > fractionDigits {
		if strings.Trim(fraction[fractionDigits:], "0") != "" {
			return 0, fmt.Errorf("decimal %q exceeds supported precision", value)
		}
		fraction = fraction[:fractionDigits]
	}
	fraction += strings.Repeat("0", fractionDigits-len(fraction))
	minor := int64(0)
	if fraction != "" {
		minor, err = strconv.ParseInt(fraction, 10, 64)
		if err != nil {
			return 0, err
		}
	}
	if whole > (math.MaxInt64-minor)/scale {
		return 0, errors.New("price overflows int64")
	}
	return whole*scale + minor, nil
}

func parseInteger(value string) (int64, error) {
	if strings.Contains(value, ".") {
		return 0, fmt.Errorf("expected integer, got %q", value)
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed < 0 {
		return 0, fmt.Errorf("invalid integer %q", value)
	}
	return parsed, nil
}

func multiplyMajor(price, quantity, scale int64) (int64, error) {
	if quantity != 0 && price > math.MaxInt64/quantity {
		return 0, errors.New("market cap overflows int64")
	}
	return price * quantity / scale, nil
}

func averageVolume(candles []domain.Candle, days int) int64 {
	if len(candles) == 0 {
		return 0
	}
	if days > len(candles) {
		days = len(candles)
	}
	var total int64
	for _, candle := range candles[:days] {
		if candle.Volume > math.MaxInt64-total {
			return math.MaxInt64
		}
		total += candle.Volume
	}
	return total / int64(days)
}

func aggregateWeekly(daily []domain.Candle) []domain.Candle {
	if len(daily) == 0 {
		return nil
	}
	ascending := append([]domain.Candle(nil), daily...)
	sort.Slice(ascending, func(i, j int) bool { return ascending[i].Timestamp.Before(ascending[j].Timestamp) })
	weekly := make([]domain.Candle, 0, len(ascending)/5+1)
	var year, week int
	for _, candle := range ascending {
		y, w := candle.Timestamp.ISOWeek()
		if len(weekly) == 0 || y != year || w != week {
			weekly = append(weekly, candle)
			year, week = y, w
			continue
		}
		current := &weekly[len(weekly)-1]
		if candle.High > current.High {
			current.High = candle.High
		}
		if candle.Low < current.Low {
			current.Low = candle.Low
		}
		current.Close = candle.Close
		current.Volume += candle.Volume
	}
	for left, right := 0, len(weekly)-1; left < right; left, right = left+1, right-1 {
		weekly[left], weekly[right] = weekly[right], weekly[left]
	}
	return weekly
}
