package util

import (
	"errors"
	"fmt"
)

type LayerError struct {
	Layer   string
	Message string
	Err     error
}

func (e *LayerError) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.Message == "" {
		return fmt.Sprintf("%s error: %v", e.Layer, e.Err)
	}
	return fmt.Sprintf("%s error: %s: %v", e.Layer, e.Message, e.Err)
}

func (e *LayerError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func WrapLayer(layer, message string, err error) error {
	if err == nil {
		return nil
	}

	var existing *LayerError
	if errors.As(err, &existing) && existing.Layer == layer {
		if message == "" || message == existing.Message {
			return err
		}

		if existing.Message == "" {
			return &LayerError{
				Layer:   layer,
				Message: message,
				Err:     existing.Err,
			}
		}

		return &LayerError{
			Layer:   layer,
			Message: message + ": " + existing.Message,
			Err:     existing.Err,
		}
	}

	return &LayerError{
		Layer:   layer,
		Message: message,
		Err:     err,
	}
}
