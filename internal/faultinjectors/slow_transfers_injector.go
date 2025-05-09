package faultinjectors

import (
	"context"
	"strconv"
	"time"

	"github.com/richardpark-msft/amqpfaultinjector/internal/logging"
	"github.com/richardpark-msft/amqpfaultinjector/internal/proto/frames"
	"github.com/richardpark-msft/amqpfaultinjector/internal/utils"
)

// NewSlowTransfersInjector creates a SlowTransferFrames injector, which slows down any incoming
// TRANSFER frames, to non-cbs/non-management links.
//   - delayForFrame controls how long we hold onto a TRANSFER frame, before forwarding the frame to the Receiver.
//   - afterNumFrames controls how many TRANSFER frames are sent before we start delaying.
func NewSlowTransfersInjector(delayForFrame time.Duration, afterNumFrames int) *SlowTransfersInjector {
	return &SlowTransfersInjector{
		delayForFrame:  delayForFrame,
		afterNumFrames: afterNumFrames,
	}
}

type SlowTransfersInjector struct {
	numTransfers int

	delayForFrame  time.Duration
	afterNumFrames int
}

func (sti *SlowTransfersInjector) Callback(ctx context.Context, params MirrorCallbackParams) ([]MetaFrame, error) {
	slogger := logging.SloggerFromContext(ctx)

	if params.Out {
		return []MetaFrame{{Action: MetaFrameActionPassthrough, Frame: params.Frame}}, nil
	}
	if params.ManagementOrCBS() {
		return []MetaFrame{{Action: MetaFrameActionPassthrough, Frame: params.Frame}}, nil
	}

	transferFrame, ok := params.Frame.Body.(*frames.PerformTransfer)

	if !ok {
		return []MetaFrame{{Action: MetaFrameActionPassthrough, Frame: params.Frame}}, nil
	}

	sti.numTransfers++

	if sti.numTransfers <= sti.afterNumFrames {
		return []MetaFrame{{Action: MetaFrameActionPassthrough, Frame: params.Frame}}, nil
	}

	deliveryID := "<unspecified>"

	if transferFrame.DeliveryID != nil {
		deliveryID = strconv.FormatInt(int64(uint64(*transferFrame.DeliveryID)), 10)
	}

	slogger.Info("Slowing down TRANSFER frame", "deliveryid", deliveryID, "more", transferFrame.More)

	// if the sleep is cancelled then the injector is being closed out, we should exit.
	if err := utils.Sleep(ctx, sti.delayForFrame); err != nil {
		return nil, err
	}

	slogger.Info("Sending TRANSFER frame", "deliveryid", deliveryID, "more", transferFrame.More)

	// In this example we're just sending the frame to the client/server without any changes, but we could
	// change the frame, add _more_ frames, or even drop the frame altogether. This is all controlled by the
	// returned slice of MetaFrames.
	return []MetaFrame{
		{Action: MetaFrameActionPassthrough, Frame: params.Frame},
	}, nil
}
