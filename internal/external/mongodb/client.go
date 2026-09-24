// Package mongodb는 저장 스키마와 독립적인 MongoDB 연결 수명주기를 관리합니다.
package mongodb

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

type Client struct {
	client           *mongo.Client
	operationTimeout time.Duration
	shutdownTimeout  time.Duration
	closeOnce        sync.Once
	closeErr         error
}

// Connect는 primary에 Ping이 성공해야 연결을 반환합니다. 실패한 클라이언트도 정리합니다.
func Connect(ctx context.Context, cfg Config) (*Client, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, safeError("connect", err)
	}
	opts := options.Client().ApplyURI(cfg.URI).
		SetConnectTimeout(cfg.ConnectTimeout).
		SetServerSelectionTimeout(cfg.ConnectTimeout).
		SetTimeout(cfg.OperationTimeout).
		SetMinPoolSize(cfg.MinPoolSize).SetMaxPoolSize(cfg.MaxPoolSize).
		SetLoggerOptions(options.Logger().SetSink(silentLogSink{}))
	if err := opts.Validate(); err != nil {
		return nil, errors.New("invalid MONGODB_URI or MongoDB client options")
	}
	raw, err := mongo.Connect(opts)
	if err != nil {
		return nil, safeError("connect", err)
	}
	client := &Client{client: raw, operationTimeout: cfg.OperationTimeout, shutdownTimeout: cfg.ShutdownTimeout}
	pingCtx, cancel := context.WithTimeout(ctx, cfg.ConnectTimeout)
	defer cancel()
	if err := client.Ping(pingCtx); err != nil {
		return nil, errors.Join(err, client.Close())
	}
	return client, nil
}

func (c *Client) Ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, c.operationTimeout)
	defer cancel()
	return safeError("ping", c.client.Ping(ctx, readpref.Primary()))
}

// Close는 시그널로 취소된 실행 컨텍스트와 독립적으로 제한 시간 내 연결을 정리합니다.
// 여러 호출자가 종료를 요청해도 드라이버의 정리는 한 번만 수행합니다.
func (c *Client) Close() error {
	c.closeOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), c.shutdownTimeout)
		defer cancel()
		c.closeErr = safeError("disconnect", c.client.Disconnect(ctx))
	})
	return c.closeErr
}

// 드라이버 오류에는 호스트, URI, 서버 응답이 포함될 수 있으므로 원본을 래핑하지 않습니다.
func safeError(operation string, err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, context.Canceled):
		return fmt.Errorf("MongoDB %s: %w", operation, context.Canceled)
	case mongo.IsTimeout(err):
		return fmt.Errorf("MongoDB %s: %w", operation, context.DeadlineExceeded)
	case errors.Is(err, mongo.ErrClientDisconnected):
		return fmt.Errorf("MongoDB %s: client disconnected", operation)
	default:
		return fmt.Errorf("MongoDB %s failed; check connectivity, authentication and TLS settings", operation)
	}
}

// 환경변수로 활성화된 드라이버 진단 로그도 연결정보를 출력하지 않도록 차단합니다.
type silentLogSink struct{}

func (silentLogSink) Info(int, string, ...any)    {}
func (silentLogSink) Error(error, string, ...any) {}
