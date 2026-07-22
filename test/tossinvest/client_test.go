// 이 파일은 외부 패키지 관점에서 토스증권 공통 클라이언트의 인증과 오류 처리를 검증합니다.
package tossinvest_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"auto-stock-trading/internal/tossinvest"
)

func TestClientAuthenticatesRequestAndAddsAccount(t *testing.T) {
	var tokenCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth2/token":
			tokenCalls.Add(1)
			if err := r.ParseForm(); err != nil {
				t.Errorf("ParseForm(): %v", err)
			}
			if r.Form.Get("grant_type") != "client_credentials" || r.Form.Get("client_id") != "client-id" || r.Form.Get("client_secret") != "client-secret" {
				t.Errorf("unexpected token form: %v", r.Form)
			}
			writeJSON(w, http.StatusOK, map[string]any{"access_token": "token-one", "token_type": "Bearer", "expires_in": 3600})
		case "/api/v1/stocks":
			if got := r.Header.Get("Authorization"); got != "Bearer token-one" {
				t.Errorf("Authorization = %q", got)
			}
			if got := r.Header.Get("X-Tossinvest-Account"); got != "7" {
				t.Errorf("account header = %q", got)
			}
			writeJSON(w, http.StatusOK, map[string]any{"result": map[string]any{}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := newClient(t, server.URL, nil)
	request, err := client.NewRequest(context.Background(), http.MethodGet, "/api/v1/stocks?symbols=005930", nil)
	if err != nil {
		t.Fatalf("NewRequest(): %v", err)
	}
	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("Do(): %v", err)
	}
	response.Body.Close()
	if tokenCalls.Load() != 1 {
		t.Fatalf("token calls = %d, want 1", tokenCalls.Load())
	}
}

func TestClientRefreshesRejectedTokenOnlyOnceForConcurrentRequests(t *testing.T) {
	var tokenCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oauth2/token" {
			call := tokenCalls.Add(1)
			writeJSON(w, http.StatusOK, map[string]any{"access_token": fmt.Sprintf("token-%d", call), "token_type": "Bearer", "expires_in": 3600})
			return
		}
		if r.Header.Get("Authorization") == "Bearer token-1" {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"error": map[string]any{"requestId": "req-expired", "code": "expired-token", "message": "expired"}})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"result": map[string]any{}})
	}))
	defer server.Close()
	client := newClient(t, server.URL, nil)

	const count = 8
	start := make(chan struct{})
	errorsSeen := make(chan error, count)
	var group sync.WaitGroup
	for range count {
		group.Add(1)
		go func() {
			defer group.Done()
			<-start
			req, err := client.NewRequest(context.Background(), http.MethodGet, "/api/v1/stocks", nil)
			if err == nil {
				var response *http.Response
				response, err = client.Do(req)
				if response != nil {
					response.Body.Close()
				}
			}
			errorsSeen <- err
		}()
	}
	close(start)
	group.Wait()
	close(errorsSeen)
	for err := range errorsSeen {
		if err != nil {
			t.Errorf("Do(): %v", err)
		}
	}
	if tokenCalls.Load() != 2 {
		t.Fatalf("token calls = %d, want 2", tokenCalls.Load())
	}
}

func TestClientReturnsStructuredAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oauth2/token" {
			writeJSON(w, http.StatusOK, map[string]any{"access_token": "token", "token_type": "Bearer", "expires_in": 3600})
			return
		}
		w.Header().Set("X-Request-Id", "header-id")
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": map[string]any{"code": "invalid-request", "message": "bad input", "data": map[string]any{"field": "symbol"}}})
	}))
	defer server.Close()
	client := newClient(t, server.URL, nil)
	req, _ := client.NewRequest(context.Background(), http.MethodGet, "/api/v1/stocks", nil)
	_, err := client.Do(req)
	var apiErr *tossinvest.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error = %T, want *APIError", err)
	}
	if apiErr.StatusCode != 400 || apiErr.Code != "invalid-request" || apiErr.RequestID != "header-id" {
		t.Fatalf("unexpected API error: %+v", apiErr)
	}
	if strings.Contains(apiErr.Error(), "bad input") {
		t.Fatal("APIError.Error() exposed server message")
	}
}

func TestClientRetriesRateLimitedGET(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oauth2/token" {
			writeJSON(w, http.StatusOK, map[string]any{"access_token": "token", "token_type": "Bearer", "expires_in": 3600})
			return
		}
		if requests.Add(1) == 1 {
			w.Header().Set("Retry-After", "0")
			writeJSON(w, http.StatusTooManyRequests, map[string]any{"error": map[string]any{"requestId": "rate", "code": "rate-limit-exceeded", "message": "slow down"}})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"result": map[string]any{}})
	}))
	defer server.Close()
	client := newClient(t, server.URL, nil)
	req, _ := client.NewRequest(context.Background(), http.MethodGet, "/api/v1/stocks", nil)
	response, err := client.Do(req)
	if err != nil {
		t.Fatalf("Do(): %v", err)
	}
	response.Body.Close()
	if requests.Load() != 2 {
		t.Fatalf("requests = %d, want 2", requests.Load())
	}
}

func TestClientDoesNotLogCredentialsOrAccount(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug}))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oauth2/token" {
			writeJSON(w, http.StatusOK, map[string]any{"access_token": "very-secret-token", "token_type": "Bearer", "expires_in": 3600})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"result": map[string]any{}})
	}))
	defer server.Close()
	client := newClient(t, server.URL, logger)
	req, _ := client.NewRequest(context.Background(), http.MethodGet, "/api/v1/stocks", nil)
	response, err := client.Do(req)
	if err != nil {
		t.Fatalf("Do(): %v", err)
	}
	response.Body.Close()
	for _, secret := range []string{"client-id", "client-secret", "very-secret-token", "\"7\""} {
		if strings.Contains(logs.String(), secret) {
			t.Fatalf("logs contain sensitive value %q: %s", secret, logs.String())
		}
	}
}

func newClient(t *testing.T, baseURL string, logger *slog.Logger) *tossinvest.Client {
	t.Helper()
	client, err := tossinvest.NewClient(tossinvest.Config{ClientID: "client-id", ClientSecret: "client-secret", Account: "7", BaseURL: baseURL, Logger: logger, MaxRetries: 1, RetryBase: time.Nanosecond, RetryMax: time.Nanosecond})
	if err != nil {
		t.Fatalf("NewClient(): %v", err)
	}
	return client
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
