package errors

import (
	"errors"
	"fmt"
)

var DefaultMessage = func(err error, msg string) error {
	return fmt.Errorf("%s: %w", msg, err)
}

func noop(err error) error {
	return err
}

func Join(errs ...error) error {
	return errors.Join(errs...)
}

func Make(msg string) error {
	return errors.New(msg)
}

func Message(err error, msg string) error {
	if disabled {
		return err
	}
	return DefaultMessage(err, msg)
}

func MessageSwap(err *error, msg string) {
	if disabled {
		return
	}
	if *err == nil {
		return
	}
	*err = DefaultMessage(*err, msg)
}

func MessageFunc(msg string) func(error) error {
	if disabled {
		return noop
	}

	return func(err error) error {
		return DefaultMessage(err, msg)
	}
}
