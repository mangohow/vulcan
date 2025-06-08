package errors

import (
	"fmt"
	"strings"
)

type ErrorChain struct {
	errors []error
}

func (e *ErrorChain) Error() string {
	builder := strings.Builder{}
	for i := len(e.errors) - 1; i >= 0; i-- {
		builder.WriteString(e.errors[i].Error())
		builder.WriteString("\n")
	}

	return builder.String()
}

func Errorf(format string, args ...any) error {
	return &ErrorChain{errors: []error{fmt.Errorf(format, args...)}}
}

func Wrapf(err error, format string, args ...any) error {
	e, ok := err.(*ErrorChain)
	if !ok {
		e = &ErrorChain{errors: []error{err}}
	}

	e.errors = append(e.errors, fmt.Errorf(format, args...))

	return e
}
