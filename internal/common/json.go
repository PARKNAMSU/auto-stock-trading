// Package common은 여러 어댑터와 도메인에서 공유하는 범용 기능을 제공합니다.
package common

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

// EncodeJSON은 값을 JSON으로 직렬화하고 재전송 가능한 메모리 기반 요청 본문을 반환합니다.
// bytes.Reader를 사용하므로 http.NewRequest가 GetBody를 구성할 수 있어 인증 및 호출 제한 재시도에 안전합니다.
func EncodeJSON(value any) (io.Reader, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("encode JSON request: %w", err)
	}
	return bytes.NewReader(data), nil
}
