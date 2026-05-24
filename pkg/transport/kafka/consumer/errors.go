package consumer

import "errors"

var (
	ErrSkip      = errors.New("skip")
	ErrRetryable = errors.New("retryable")
	ErrFatal     = errors.New("fatal")
)

func Skip(err error) error {
	return errors.Join(ErrSkip, err)
}

func Retryable(err error) error {
	return errors.Join(ErrRetryable, err)
}

func Fatal(err error) error {
	return errors.Join(ErrFatal, err)
}
