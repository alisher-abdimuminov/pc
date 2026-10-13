// Package apperr holds the typed errors that become JSON error responses.
package apperr

import "net/http"

type Error struct {
	Status  int
	Code    string
	Message string
	Fields  map[string]string
}

func (e *Error) Error() string { return e.Code + ": " + e.Message }

func New(status int, code, msg string) *Error {
	return &Error{Status: status, Code: code, Message: msg}
}

func NotFound() *Error {
	return New(http.StatusNotFound, "not_found", "Topilmadi")
}

func Unauthorized() *Error {
	return New(http.StatusUnauthorized, "unauthorized", "Avtorizatsiyadan o'tilmagan")
}

func Forbidden() *Error {
	return New(http.StatusForbidden, "forbidden", "Ruxsat yo'q")
}

func Validation(msg string, fields map[string]string) *Error {
	return &Error{Status: http.StatusUnprocessableEntity, Code: "validation_error", Message: msg, Fields: fields}
}
