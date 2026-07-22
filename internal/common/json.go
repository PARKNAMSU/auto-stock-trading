// Package common은 여러 어댑터와 도메인에서 공유하는 범용 기능을 제공합니다.
package common

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

// EncodeJSON returns a replayable JSON request body.
func EncodeJSON(value any) (io.Reader, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("encode JSON request: %w", err)
	}
	return bytes.NewReader(data), nil
}
