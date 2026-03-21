package auth

import (
	"encoding/hex"
	"testing"
)

// TestHashAndVerifyPassword 用于单元测试。
func TestHashAndVerifyPassword(t *testing.T) {
	t.Parallel()

	hash, err := HashPassword("pass-123")
	if err != nil {
		t.Fatalf("HashPassword error: %v", err)
	}
	if hash == "" {
		t.Fatalf("expected non-empty hash")
	}
	if !VerifyPassword("pass-123", hash) {
		t.Fatalf("expected password verify success")
	}
	if VerifyPassword("wrong-pass", hash) {
		t.Fatalf("expected password verify fail for wrong password")
	}
}

// TestNewToken 用于单元测试。
func TestNewToken(t *testing.T) {
	t.Parallel()

	tokenA, err := NewToken()
	if err != nil {
		t.Fatalf("NewToken A error: %v", err)
	}
	tokenB, err := NewToken()
	if err != nil {
		t.Fatalf("NewToken B error: %v", err)
	}

	if len(tokenA) != 64 || len(tokenB) != 64 {
		t.Fatalf("unexpected token length: %d %d", len(tokenA), len(tokenB))
	}
	if _, err := hex.DecodeString(tokenA); err != nil {
		t.Fatalf("tokenA is not valid hex: %v", err)
	}
	if _, err := hex.DecodeString(tokenB); err != nil {
		t.Fatalf("tokenB is not valid hex: %v", err)
	}
	if tokenA == tokenB {
		t.Fatalf("expected two generated tokens to be different")
	}
}
