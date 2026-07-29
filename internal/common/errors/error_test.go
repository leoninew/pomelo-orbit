package apperror

import (
	"errors"
	"net/http"
	"testing"
)

func TestClassify(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want Classification
	}{
		{name: "validation", err: New(KindValidation, "Invalid JSON body"), want: Classification{StatusCode: http.StatusBadRequest, Code: "validation_failed", Message: "Invalid JSON body"}},
		{name: "unauthorized", err: New(KindUnauthorized, ""), want: Classification{StatusCode: http.StatusUnauthorized, Code: "unauthorized", Message: "Unauthorized."}},
		{name: "forbidden", err: New(KindForbidden, ""), want: Classification{StatusCode: http.StatusForbidden, Code: "forbidden", Message: "Forbidden."}},
		{name: "not found", err: New(KindNotFound, ""), want: Classification{StatusCode: http.StatusNotFound, Code: "not_found", Message: "Resource not found."}},
		{name: "conflict", err: New(KindConflict, ""), want: Classification{StatusCode: http.StatusConflict, Code: "conflict", Message: "Conflict."}},
		{name: "method not allowed", err: New(KindMethodNotAllowed, ""), want: Classification{StatusCode: http.StatusMethodNotAllowed, Code: "method_not_allowed", Message: "Method not allowed."}},
		{name: "rate limited", err: New(KindRateLimited, ""), want: Classification{StatusCode: http.StatusTooManyRequests, Code: "rate_limited", Message: "Too many requests."}},
		{name: "service unavailable", err: New(KindUnavailable, ""), want: Classification{StatusCode: http.StatusServiceUnavailable, Code: "service_unavailable", Message: "Service unavailable."}},
		{name: "internal ignores message and cause", err: Wrap(KindInternal, "database password", errors.New("token=secret")), want: Classification{StatusCode: http.StatusInternalServerError, Code: "internal_error", Message: "Internal server error."}},
		{name: "unknown error", err: errors.New("database password"), want: Classification{StatusCode: http.StatusInternalServerError, Code: "internal_error", Message: "Internal server error."}},
		{name: "unknown kind ignores custom code", err: NewWithCode(Kind("unexpected"), "unexpected_error", "not for output"), want: Classification{StatusCode: http.StatusInternalServerError, Code: "internal_error", Message: "Internal server error."}},
		{name: "valid custom code", err: NewWithCode(KindConflict, "version_already_published", "Version already published."), want: Classification{StatusCode: http.StatusConflict, Code: "version_already_published", Message: "Version already published."}},
		{name: "invalid custom code", err: NewWithCode(KindConflict, "VersionAlreadyPublished", "Version already published."), want: Classification{StatusCode: http.StatusConflict, Code: "conflict", Message: "Version already published."}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Classify(tc.err); got != tc.want {
				t.Fatalf("unexpected classification: got %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestAsSupportsWrappedPointerError(t *testing.T) {
	original := &Error{Kind: KindNotFound, Message: "Project not found"}
	wrapped := errors.Join(errors.New("context"), original)

	appErr, ok := As(wrapped)
	if !ok || appErr != *original {
		t.Fatalf("unexpected application error: got %+v, found=%t", appErr, ok)
	}
}
