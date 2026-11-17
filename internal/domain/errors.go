package domain

import "errors"

var ErrNotFound = errors.New("resource not found")
var ErrConflict = errors.New("resource conflict")
