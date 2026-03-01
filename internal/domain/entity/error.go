package entity

import "errors"

var ErrInvalidQuery = errors.New("invalid query")
var ErrInvalidType = errors.New("invalid type")

var ErrPackageNotFound = errors.New("package not found")
var ErrBranchNotFound = errors.New("branch not found")

type ValidationError struct {
	Err    error
	Field  string
	Reason string
}

func (e ValidationError) Error() string {
	return e.Err.Error()
}

func (e ValidationError) Unwrap() error {
	return e.Err
}
