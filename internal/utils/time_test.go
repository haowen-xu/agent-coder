package utils

import (
	"testing"
	"time"
)

// TestNowUTC 用于单元测试。
func TestNowUTC(t *testing.T) {
	before := time.Now().UTC()
	got := NowUTC()
	after := time.Now().UTC()

	if got.Location() != time.UTC {
		t.Fatalf("NowUTC location should be UTC, got=%v", got.Location())
	}
	if got.Before(before.Add(-1*time.Second)) || got.After(after.Add(1*time.Second)) {
		t.Fatalf("NowUTC out of expected range: got=%s before=%s after=%s", got, before, after)
	}
}
