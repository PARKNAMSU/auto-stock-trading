// 이 파일은 외부 패키지 관점에서 공용 JSON 인코딩 기능을 검증합니다.
package common_test

import (
	"io"
	"strings"
	"testing"

	"auto-stock-trading/internal/common"
)

func TestEncodeJSON(t *testing.T) {
	body, err := common.EncodeJSON(struct {
		Symbol string `json:"symbol"`
	}{Symbol: "005930"})
	if err != nil {
		t.Fatalf("EncodeJSON(): %v", err)
	}

	encoded, err := io.ReadAll(body)
	if err != nil {
		t.Fatalf("ReadAll(): %v", err)
	}
	if got, want := string(encoded), `{"symbol":"005930"}`; got != want {
		t.Fatalf("encoded JSON = %q, want %q", got, want)
	}
}

func TestEncodeJSONWrapsMarshalError(t *testing.T) {
	_, err := common.EncodeJSON(make(chan int))
	if err == nil {
		t.Fatal("EncodeJSON() accepted an unsupported value")
	}
	if !strings.Contains(err.Error(), "encode JSON request") {
		t.Fatalf("error = %q", err)
	}
}
