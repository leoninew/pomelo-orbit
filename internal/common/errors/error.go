package apperror

import (
	"errors"
	"net/http"
)

type Kind string

const (
	KindValidation       Kind = "validation"
	KindUnauthorized     Kind = "unauthorized"
	KindForbidden        Kind = "forbidden"
	KindNotFound         Kind = "not_found"
	KindConflict         Kind = "conflict"
	KindMethodNotAllowed Kind = "method_not_allowed"
	KindUnavailable      Kind = "unavailable"
	KindInternal         Kind = "internal"
)

type Error struct {
	Kind    Kind
	Code    string
	Message string
	Err     error
}

type Classification struct {
	StatusCode int
	Code       string
	Message    string
}

func New(kind Kind, message string) Error {
	return Error{Kind: kind, Message: message}
}

func Wrap(kind Kind, message string, err error) Error {
	return Error{Kind: kind, Message: message, Err: err}
}

func NewWithCode(kind Kind, code string, message string) Error {
	return Error{Kind: kind, Code: code, Message: message}
}

func WrapWithCode(kind Kind, code string, message string, err error) Error {
	return Error{Kind: kind, Code: code, Message: message, Err: err}
}

func (e Error) Error() string {
	switch {
	case e.Message != "" && e.Err != nil:
		// Preserve the root cause for server-side logs.
		return e.Message + ": " + e.Err.Error()
	case e.Message != "":
		return e.Message
	case e.Err != nil:
		return e.Err.Error()
	default:
		return string(e.Kind)
	}
}

func (e Error) Unwrap() error {
	return e.Err
}

func IsKind(err error, kind Kind) bool {
	appErr, ok := As(err)
	return ok && appErr.Kind == kind
}

func StatusCode(err error) int {
	return Classify(err).StatusCode
}

func Classify(err error) Classification {
	appErr, ok := As(err)
	if !ok {
		return classificationForKind(KindInternal, "")
	}

	classification := classificationForKind(appErr.Kind, appErr.Message)
	if isPublicKind(appErr.Kind) && validCode(appErr.Code) {
		classification.Code = appErr.Code
	}
	return classification
}

func As(err error) (Error, bool) {
	var appErr Error
	if errors.As(err, &appErr) {
		return appErr, true
	}
	var appErrPtr *Error
	if errors.As(err, &appErrPtr) && appErrPtr != nil {
		return *appErrPtr, true
	}
	return Error{}, false
}

func NewForHTTPStatus(status int, message string) Error {
	return New(kindForHTTPStatus(status), message)
}

func classificationForKind(kind Kind, message string) Classification {
	var classification Classification
	switch kind {
	case KindValidation:
		classification = Classification{StatusCode: http.StatusBadRequest, Code: "validation_failed", Message: "Invalid request."}
	case KindUnauthorized:
		classification = Classification{StatusCode: http.StatusUnauthorized, Code: "unauthorized", Message: "Unauthorized."}
	case KindForbidden:
		classification = Classification{StatusCode: http.StatusForbidden, Code: "forbidden", Message: "Forbidden."}
	case KindNotFound:
		classification = Classification{StatusCode: http.StatusNotFound, Code: "not_found", Message: "Resource not found."}
	case KindConflict:
		classification = Classification{StatusCode: http.StatusConflict, Code: "conflict", Message: "Conflict."}
	case KindMethodNotAllowed:
		classification = Classification{StatusCode: http.StatusMethodNotAllowed, Code: "method_not_allowed", Message: "Method not allowed."}
	case KindUnavailable:
		classification = Classification{StatusCode: http.StatusServiceUnavailable, Code: "service_unavailable", Message: "Service unavailable."}
	default:
		return Classification{StatusCode: http.StatusInternalServerError, Code: "internal_error", Message: "Internal server error."}
	}
	if message != "" {
		classification.Message = message
	}
	return classification
}

func kindForHTTPStatus(status int) Kind {
	switch status {
	case http.StatusBadRequest:
		return KindValidation
	case http.StatusUnauthorized:
		return KindUnauthorized
	case http.StatusForbidden:
		return KindForbidden
	case http.StatusNotFound:
		return KindNotFound
	case http.StatusConflict:
		return KindConflict
	case http.StatusMethodNotAllowed:
		return KindMethodNotAllowed
	case http.StatusServiceUnavailable:
		return KindUnavailable
	default:
		return KindInternal
	}
}

func validCode(code string) bool {
	if len(code) == 0 || code[0] < 'a' || code[0] > 'z' {
		return false
	}
	for index := 1; index < len(code); index++ {
		character := code[index]
		if (character < 'a' || character > 'z') && (character < '0' || character > '9') && character != '_' {
			return false
		}
	}
	return true
}

func isPublicKind(kind Kind) bool {
	switch kind {
	case KindValidation, KindUnauthorized, KindForbidden, KindNotFound, KindConflict, KindMethodNotAllowed, KindUnavailable:
		return true
	default:
		return false
	}
}
