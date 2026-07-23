package apperror

import (
	"errors"
	"strings"
	"testing"
)

func TestErrorStringIncludesCause(t *testing.T) {
	err := Wrap(KindInternal, "Failed to delete application", errors.New("FOREIGN KEY constraint failed"))
	got := err.Error()
	if !strings.Contains(got, "Failed to delete application") {
		t.Fatalf("expected message in %q", got)
	}
	if !strings.Contains(got, "FOREIGN KEY constraint failed") {
		t.Fatalf("expected cause in %q", got)
	}
}

func TestErrorStringWithoutCause(t *testing.T) {
	err := New(KindValidation, "应用正在运行中, 请先停止后再删除")
	if got := err.Error(); got != "应用正在运行中, 请先停止后再删除" {
		t.Fatalf("got %q", got)
	}
}
