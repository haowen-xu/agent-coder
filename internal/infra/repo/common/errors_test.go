package common

import (
	"errors"
	"testing"
)

// TestErrNeedHumanMergeError 用于单元测试。
func TestErrNeedHumanMergeError(t *testing.T) {
	t.Parallel()

	var nilErr *ErrNeedHumanMerge
	if got := nilErr.Error(); got != "need human merge" {
		t.Fatalf("nil error text mismatch: %q", got)
	}

	if got := (&ErrNeedHumanMerge{}).Error(); got != "need human merge" {
		t.Fatalf("empty error text mismatch: %q", got)
	}
	if got := (&ErrNeedHumanMerge{Reason: "x", StatusCode: 409}).Error(); got != "x (status=409)" {
		t.Fatalf("status-only error text mismatch: %q", got)
	}
	if got := (&ErrNeedHumanMerge{Reason: "x", Provider: "gitlab"}).Error(); got != "x provider=gitlab" {
		t.Fatalf("provider-only error text mismatch: %q", got)
	}
	if got := (&ErrNeedHumanMerge{Reason: "x", Provider: "gitlab", StatusCode: 409}).Error(); got != "x provider=gitlab status=409" {
		t.Fatalf("provider+status error text mismatch: %q", got)
	}
}

// TestIsNeedHumanMerge 用于单元测试。
func TestIsNeedHumanMerge(t *testing.T) {
	t.Parallel()

	if IsNeedHumanMerge(nil) {
		t.Fatalf("nil should not be need human merge")
	}
	if IsNeedHumanMerge(errors.New("other")) {
		t.Fatalf("plain error should not match")
	}
	target := &ErrNeedHumanMerge{Provider: "gitlab", StatusCode: 409}
	if !IsNeedHumanMerge(target) {
		t.Fatalf("ErrNeedHumanMerge should match")
	}
	if !IsNeedHumanMerge(errors.Join(errors.New("wrap"), target)) {
		t.Fatalf("wrapped ErrNeedHumanMerge should match")
	}
}
