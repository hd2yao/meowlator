package domain

import "errors"

var (
	ErrNotFound     = errors.New("not found")
	ErrBadRequest   = errors.New("bad request")
	ErrConflict     = errors.New("conflict")
	ErrUpstream     = errors.New("upstream error")
	ErrUnauthorized = errors.New("unauthorized")
)
