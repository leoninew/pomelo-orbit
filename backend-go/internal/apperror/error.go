package apperror

import "errors"

type Kind string

const (
	KindValidation   Kind = "validation"
	KindUnauthorized Kind = "unauthorized"
	KindForbidden    Kind = "forbidden"
	KindNotFound     Kind = "not_found"
	KindConflict     Kind = "conflict"
	KindInternal     Kind = "internal"
)

type Error struct {
	Kind    Kind
	Message string
	Err     error
}

func New(kind Kind, message string) Error {
	return Error{Kind: kind, Message: message}
}

func Wrap(kind Kind, message string, err error) Error {
	return Error{Kind: kind, Message: message, Err: err}
}

func (e Error) Error() string {
	if e.Message != "" {
		return e.Message
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return string(e.Kind)
}

func (e Error) Unwrap() error {
	return e.Err
}

func IsKind(err error, kind Kind) bool {
	var appErr Error
	return errors.As(err, &appErr) && appErr.Kind == kind
}
