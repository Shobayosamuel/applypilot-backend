// Package apperr defines typed application errors that services return and
// the HTTP layer maps to status codes.
package apperr

import (
	"errors"
	"fmt"
)

type Kind int

const (
	KindInternal Kind = iota
	KindInvalid
	KindUnauthorized
	KindNotFound
	KindConflict
	KindRateLimited
	KindUnavailable
	KindUpstream
)

type Error struct {
	Kind Kind
	Msg  string
	Err  error
}

func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Msg, e.Err)
	}
	return e.Msg
}

func (e *Error) Unwrap() error { return e.Err }

func Invalid(msg string) error      { return &Error{Kind: KindInvalid, Msg: msg} }
func Unauthorized(msg string) error { return &Error{Kind: KindUnauthorized, Msg: msg} }
func NotFound(msg string) error     { return &Error{Kind: KindNotFound, Msg: msg} }
func Conflict(msg string) error     { return &Error{Kind: KindConflict, Msg: msg} }
func RateLimited(msg string) error  { return &Error{Kind: KindRateLimited, Msg: msg} }
func Unavailable(msg string, err error) error {
	return &Error{Kind: KindUnavailable, Msg: msg, Err: err}
}
func Upstream(msg string, err error) error { return &Error{Kind: KindUpstream, Msg: msg, Err: err} }

// KindOf returns the error's kind, or KindInternal for untyped errors.
func KindOf(err error) Kind {
	var e *Error
	if errors.As(err, &e) {
		return e.Kind
	}
	return KindInternal
}

// Message returns the client-safe message for typed errors.
func Message(err error) string {
	var e *Error
	if errors.As(err, &e) {
		return e.Msg
	}
	return "internal error"
}
