// Package apperr defines typed domain errors for the Wonderfeed host CLI.
package apperr

import (
	"errors"
	"fmt"
)

// Code is a stable machine-readable error class.
type Code string

const (
	// CodeInvalid is a bad operator input or configuration.
	CodeInvalid Code = "invalid"
	// CodeNotFound is a missing path or resource.
	CodeNotFound Code = "not_found"
	// CodeUnavailable is a dependency that is down (Docker, provider HTTP).
	CodeUnavailable Code = "unavailable"
	// CodeFailed is a generic failed operation.
	CodeFailed Code = "failed"
)

// Error is a typed host error with a stable code and layer op.
type Error struct {
	Code    Code
	Op      string
	Message string
	Fields  map[string]string
	Err     error
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Err == nil {
		return fmt.Sprintf("%s: %s", e.Op, e.Message)
	}
	return fmt.Sprintf("%s: %s: %v", e.Op, e.Message, e.Err)
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// With attaches a diagnostic field.
func (e *Error) With(key, value string) *Error {
	if e.Fields == nil {
		e.Fields = map[string]string{}
	}
	e.Fields[key] = value
	return e
}

// New builds a leaf domain error.
func New(code Code, op, message string) *Error {
	return &Error{Code: code, Op: op, Message: message}
}

// Wrap converts a cause into a domain error.
func Wrap(err error, code Code, op, message string) *Error {
	if err == nil {
		return nil
	}
	var existing *Error
	if errors.As(err, &existing) {
		return &Error{Code: code, Op: op, Message: message, Err: err}
	}
	return &Error{Code: code, Op: op, Message: message, Err: err}
}
