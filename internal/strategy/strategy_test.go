// 이 파일은 런타임 전략 선택과 전략 인터페이스 구현을 검증합니다.
package strategy

import "testing"

func TestNew(t *testing.T) {
	tests := []struct {
		kind Kind
	}{
		{kind: KindHold},
		{kind: KindShortTerm},
		{kind: KindMediumTerm},
		{kind: KindLongTerm},
	}

	for _, tt := range tests {
		t.Run(string(tt.kind), func(t *testing.T) {
			if _, err := New(tt.kind); err != nil {
				t.Fatalf("New(%q) returned an error: %v", tt.kind, err)
			}
		})
	}
}

func TestNewRejectsUnknownStrategy(t *testing.T) {
	if _, err := New("unknown"); err == nil {
		t.Fatal("New() accepted an unknown strategy")
	}
}

var (
	_ Strategy = (*ShortTerm)(nil)
	_ Strategy = (*MediumTerm)(nil)
	_ Strategy = (*LongTerm)(nil)
)
