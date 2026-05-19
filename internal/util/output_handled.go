package util

import "errors"

type OutputHandledError struct {
	Err error
}

func (e *OutputHandledError) Error() string {
	if e == nil || e.Err == nil {
		return ""
	}
	return e.Err.Error()
}

func (e *OutputHandledError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func MarkOutputHandled(err error) error {
	if err == nil || IsOutputHandled(err) {
		return err
	}
	return &OutputHandledError{Err: err}
}

func IsOutputHandled(err error) bool {
	var handled *OutputHandledError
	return errors.As(err, &handled)
}
