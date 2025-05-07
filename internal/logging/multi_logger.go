package logging

import "errors"

type MultiLogger struct {
	Loggers []Logger
}

func (mpl *MultiLogger) AddPacket(out bool, packet []byte) error {
	var errs []error

	for _, l := range mpl.Loggers {
		if err := l.AddPacket(out, packet); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

func (mpl *MultiLogger) Close() error {
	var errs []error

	for _, l := range mpl.Loggers {
		if err := l.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

var _ Logger = &MultiLogger{}
