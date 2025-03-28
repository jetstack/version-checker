package errors

import (
	"errors"
	"fmt"
)

type ErrorVersionNotFound struct {
	error
}

func NewVersionErrorNotFound(format string, a ...interface{}) *ErrorVersionNotFound {
	if len(a) == 0 {
		return &ErrorVersionNotFound{errors.New(format)}
	}

	return &ErrorVersionNotFound{fmt.Errorf(format, a...)}
}

// The function IsNoVersionFound checks if the error is of type
// ErrorVersionNotFound.
func IsNoVersionFound(err error) bool {
	var notFound *ErrorVersionNotFound
	return errors.As(err, &notFound)
}
