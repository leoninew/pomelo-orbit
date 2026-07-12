package sqlcommon

import (
	"database/sql"
	"errors"
	"fmt"
	"testing"

	"gitee.com/leoninew/PomeloOrbit-go/internal/repository"
)

func TestTranslateErrorMapsNoRowsToErrNotFound(t *testing.T) {
	err := TranslateError(sql.ErrNoRows)

	if !errors.Is(err, repository.ErrNotFound) {
		t.Fatal("expected repository not-found error")
	}
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatal("expected original SQL no-rows cause to be preserved")
	}
	if err.Error() != sql.ErrNoRows.Error() {
		t.Fatalf("expected error text %q, got %q", sql.ErrNoRows, err)
	}
}

func TestTranslateErrorPreservesNonNoRowsError(t *testing.T) {
	original := fmt.Errorf("query failed: %w", errors.New("connection lost"))

	if err := TranslateError(original); err != original {
		t.Fatalf("expected original error to be returned, got %v", err)
	}
}
