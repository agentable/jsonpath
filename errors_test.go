package jsonpath

import (
	"errors"
	"fmt"
	"testing"
)

func TestErrPathParse(t *testing.T) {
	t.Parallel()

	if ErrPathParse == nil {
		t.Fatal("ErrPathParse should not be nil")
	}
	if got := ErrPathParse.Error(); got != "jsonpath: parse error" {
		t.Fatalf("ErrPathParse.Error() = %q, want %q", got, "jsonpath: parse error")
	}
}

func TestErrFunction(t *testing.T) {
	t.Parallel()

	if ErrFunction == nil {
		t.Fatal("ErrFunction should not be nil")
	}
	if got := ErrFunction.Error(); got != "jsonpath: function error" {
		t.Fatalf("ErrFunction.Error() = %q, want %q", got, "jsonpath: function error")
	}
}

func TestSentinelErrorsWrapping(t *testing.T) {
	t.Parallel()

	wrapped := fmt.Errorf("bad expression: %w", ErrPathParse)
	if !errors.Is(wrapped, ErrPathParse) {
		t.Fatal("wrapped error should match ErrPathParse via errors.Is")
	}

	wrapped = fmt.Errorf("length() failed: %w", ErrFunction)
	if !errors.Is(wrapped, ErrFunction) {
		t.Fatal("wrapped error should match ErrFunction via errors.Is")
	}
}

func TestSentinelErrorsDistinct(t *testing.T) {
	t.Parallel()

	if errors.Is(ErrPathParse, ErrFunction) {
		t.Fatal("ErrPathParse and ErrFunction should be distinct")
	}
}
