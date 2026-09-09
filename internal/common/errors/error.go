package apperror

import (
	"errors"
)

type Kind string

const (
	KindValidation       Kind = "validation"
	KindUnauthorized     Kind = "unauthorized"
	KindForbidden        Kind = "forbidden"
	KindNotFound         Kind = "not_found"
	KindConflict         Kind = "conflict"
	KindMethodNotAllowed Kind = "method_not_allowed"
	KindRateLimited      Kind = "rate_limited" // use only when the product has 429 semantics
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
	Code    string
	Message string
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

func classificationForKind(kind Kind, message string) Classification {
	var classification Classification
	switch kind {
	case KindValidation:
		classification = Classification{Code: "validation_failed", Message: "Invalid request."}
	case KindUnauthorized:
		classification = Classification{Code: "unauthorized", Message: "Unauthorized."}
	case KindForbidden:
		classification = Classification{Code: "forbidden", Message: "Forbidden."}
	case KindNotFound:
		classification = Classification{Code: "not_found", Message: "Resource not found."}
	case KindConflict:
		classification = Classification{Code: "conflict", Message: "Conflict."}
	case KindMethodNotAllowed:
		classification = Classification{Code: "method_not_allowed", Message: "Method not allowed."}
	case KindRateLimited:
		classification = Classification{Code: "rate_limited", Message: "Too many requests."}
	case KindUnavailable:
		classification = Classification{Code: "service_unavailable", Message: "Service unavailable."}
	default:
		return Classification{Code: "internal_error", Message: "Internal server error."}
	}
	if message != "" {
		classification.Message = message
	}
	return classification
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
	case KindValidation, KindUnauthorized, KindForbidden, KindNotFound, KindConflict,
		KindMethodNotAllowed, KindRateLimited, KindUnavailable:
		return true
	default:
		return false
	}
}
