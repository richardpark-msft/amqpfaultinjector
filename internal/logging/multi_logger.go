package logging

import "errors"

type MultiLogger struct {
	Loggers []Logger
}

// TODO: inefficient error handling...

func (mpl *MultiLogger) AddPacket(out bool, packet []byte) error {
	var errs []error

	for _, l := range mpl.Loggers {
		errs = append(errs, l.AddPacket(out, packet))
	}

	return errors.Join(errs...)
}

func (mpl *MultiLogger) Close() error {
	var errs []error

	for _, l := range mpl.Loggers {
		errs = append(errs, l.Close())
	}

	return errors.Join(errs...)
}

var _ Logger = &MultiLogger{}
