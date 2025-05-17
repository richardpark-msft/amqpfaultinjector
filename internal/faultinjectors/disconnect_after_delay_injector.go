package faultinjectors

import (
	"context"
	"time"

	"github.com/richardpark-msft/amqpfaultinjector/internal/logging"
	"github.com/richardpark-msft/amqpfaultinjector/internal/proto/frames"
)

// DisconnectAfterDelayInjector disconnects the entire connection after a specified delay.
type DisconnectAfterDelayInjector struct {
	Delay time.Duration
}

// NewDisconnectAfterDelayInjector creates a new DisconnectAfterDelayInjector with the given delay.
func NewDisconnectAfterDelayInjector(delay time.Duration) *DisconnectAfterDelayInjector {
	return &DisconnectAfterDelayInjector{Delay: delay}
}

func (d *DisconnectAfterDelayInjector) Callback(ctx context.Context, params MirrorCallbackParams) ([]MetaFrame, error) {
	slogger := logging.SloggerFromContext(ctx)

	// send the delay after the first non-mgmt or non-cbs link has been created.
	if !params.Out && !params.ManagementOrCBS() && params.Type() == frames.BodyTypeAttach {
		slogger.Info("Disconnect scheduled", "delay", d.Delay)

		return []MetaFrame{
			{
				Action: MetaFrameActionPassthrough,
				Frame:  params.Frame,
			},
			{
				Action: MetaFrameActionDisconnect,
				Delay:  d.Delay,
				Frame: &frames.Frame{
					Body: &frames.EmptyFrame{},
				},
			},
		}, nil
	}

	return []MetaFrame{
		{
			Action: MetaFrameActionPassthrough,
			Frame:  params.Frame,
		},
	}, nil
}
