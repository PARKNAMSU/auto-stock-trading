package mongodb_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"auto-stock-trading/internal/external/mongodb"
)

func settings(uri string) mongodb.Config {
	return mongodb.Config{URI: uri, ConnectTimeout: 150 * time.Millisecond, OperationTimeout: time.Second, ShutdownTimeout: time.Second, MaxPoolSize: 2}
}

func TestConfigFormattingRedactsURI(t *testing.T) {
	cfg := settings("mongodb://test-user:secret-marker@private-host:27017")
	var log bytes.Buffer
	slog.New(slog.NewJSONHandler(&log, nil)).Info("settings", "mongodb", cfg)
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	for _, output := range []string{fmt.Sprintf("%v %+v %#v", cfg, cfg, cfg), string(data), log.String()} {
		for _, secret := range []string{"secret-marker", "private-host", "test-user"} {
			if strings.Contains(output, secret) {
				t.Fatal("connection settings leaked")
			}
		}
	}
}

func TestRejectsInvalidURIWithoutLeaking(t *testing.T) {
	for _, uri := range []string{"https://secret-marker", "mongodb://test-user:secret-marker@", "mongodb://127.0.0.1/?connectTimeoutMS=secret-marker"} {
		client, err := mongodb.Connect(context.Background(), settings(uri))
		if err == nil || client != nil {
			t.Fatal("invalid URI accepted")
		}
		if strings.Contains(err.Error(), "secret-marker") || strings.Contains(err.Error(), "test-user") {
			t.Fatal("URI leaked in error")
		}
	}
}

func TestConnectRefusedIsBounded(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	client, err := mongodb.Connect(context.Background(), settings("mongodb://"+address))
	if client != nil || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected timeout, got %v", err)
	}
	if time.Since(start) > 2*time.Second {
		t.Fatal("connect exceeded timeout budget")
	}
	if strings.Contains(err.Error(), address) {
		t.Fatal("endpoint leaked")
	}
}

func TestConnectCancellationAndStalledHandshake(t *testing.T) {
	for _, cancelDuringConnect := range []bool{false, true} {
		t.Run(fmt.Sprint(cancelDuringConnect), func(t *testing.T) {
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			defer listener.Close()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if cancelDuringConnect {
				go func() {
					conn, err := listener.Accept()
					if err == nil {
						defer conn.Close()
						cancel()
					}
				}()
			}
			// TCP 연결은 수락되지만 MongoDB handshake 응답은 오지 않는 서버입니다.
			start := time.Now()
			client, err := mongodb.Connect(ctx, settings("mongodb://"+listener.Addr().String()))
			want := context.DeadlineExceeded
			if cancelDuringConnect {
				want = context.Canceled
			}
			if client != nil || !errors.Is(err, want) {
				t.Fatalf("expected %v, got %v", want, err)
			}
			if time.Since(start) > 2*time.Second {
				t.Fatal("connect did not stop promptly")
			}
		})
	}
}

func TestConnectAlreadyCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	client, err := mongodb.Connect(ctx, settings("mongodb://127.0.0.1:27018"))
	if client != nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation, got %v", err)
	}
}

// 통합 테스트는 별도의 일회성 MongoDB에 연결하며 데이터/컬렉션을 생성하지 않습니다.
func TestIntegrationLifecycle(t *testing.T) {
	uri := os.Getenv("MONGODB_INTEGRATION_URI")
	if uri == "" {
		t.Skip("set MONGODB_INTEGRATION_URI to run against a disposable MongoDB")
	}
	cfg := settings(uri)
	cfg.ConnectTimeout = 5 * time.Second
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	client, err := mongodb.Connect(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = client.Close() })
	if err := client.Ping(ctx); err != nil {
		t.Fatal(err)
	}
	cancel()
	if err := client.Ping(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected canceled ping, got %v", err)
	}
	start := time.Now()
	var closers sync.WaitGroup
	for range 4 {
		closers.Go(func() {
			if err := client.Close(); err != nil {
				t.Errorf("concurrent close: %v", err)
			}
		})
	}
	closers.Wait()
	if time.Since(start) > 2*time.Second {
		t.Fatal("shutdown exceeded timeout budget")
	}
	if err := client.Ping(context.Background()); err == nil {
		t.Fatal("ping succeeded after disconnect")
	}
	if err := client.Close(); err != nil {
		t.Fatalf("repeated close: %v", err)
	}
}
