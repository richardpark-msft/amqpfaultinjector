package faultinjectors

import (
	"context"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/richardpark-msft/amqpfaultinjector/internal/proto"
	"github.com/richardpark-msft/amqpfaultinjector/internal/proto/frames"
	"github.com/stretchr/testify/require"
)

func TestSlowTransfersInjector(t *testing.T) {
	const after = 3
	sfi := NewSlowTransfersInjector(time.Second, after)

	sm := loadStateMap(t)

	params := MirrorCallbackParams{
		Out: false,
		Frame: &frames.Frame{
			Header: frames.Header{Channel: 0},
			Body: &frames.PerformTransfer{
				Handle: 0,
			},
		},
		StateMap: sm,
	}

	for i := range after {
		now := time.Now()
		values, err := sfi.Callback(context.Background(), params)
		duration := time.Since(now)

		require.NoError(t, err, "Iteration: %d", i)
		require.NotEmpty(t, values, "Iteration: %d", i)
		// a little wiggle room here on the sleep, but this should prove that we're not
		// doing the TRANSFER frame pause.
		require.Greater(t, 500*time.Millisecond, duration, "Iteration: %d", i)
		require.Equal(t, i+1, sfi.numTransfers)
	}

	for i := 0; i < 2; i++ {
		now := time.Now()

		values, err := sfi.Callback(context.Background(), params)
		require.NoError(t, err)
		require.NotEmpty(t, values)

		require.LessOrEqual(t, time.Second, time.Since(now))
	}
}

func TestSlowTransfersInjector_Passthrough(t *testing.T) {
	t.Run("outbound frames ignored", func(t *testing.T) {
		sfi := NewSlowTransfersInjector(time.Second, 1)

		sm := proto.NewStateMap()
		params := MirrorCallbackParams{Out: true, Frame: &frames.Frame{}, StateMap: sm}

		values, err := sfi.Callback(context.Background(), params)
		require.NoError(t, err)
		require.Equal(t, MetaFrameActionPassthrough, values[0].Action)
		require.Same(t, params.Frame, values[0].Frame)
		require.Equal(t, 0, sfi.numTransfers)
	})

	t.Run("$cbs ignored", func(t *testing.T) {
		sfi := NewSlowTransfersInjector(time.Second, 1)

		sm := loadCBSStateMap(t)

		fr := &frames.Frame{
			Header: frames.Header{Channel: 0},
			Body: &frames.PerformTransfer{
				Handle: 1,
			},
		}

		params := MirrorCallbackParams{Out: false, Frame: fr, StateMap: sm}

		values, err := sfi.Callback(context.Background(), params)
		require.NoError(t, err)
		require.Equal(t, MetaFrameActionPassthrough, values[0].Action)
		require.Same(t, params.Frame, values[0].Frame)
		require.Equal(t, 0, sfi.numTransfers)
	})

	t.Run("non-transfer frames", func(t *testing.T) {
		sfi := NewSlowTransfersInjector(time.Second, 1)

		sm := loadStateMap(t)

		fr := &frames.Frame{
			Header: frames.Header{Channel: 0},
			Body: &frames.PerformFlow{
				Handle: to.Ptr[uint32](0),
			},
		}

		params := MirrorCallbackParams{Out: false, Frame: fr, StateMap: sm}

		values, err := sfi.Callback(context.Background(), params)
		require.NoError(t, err)
		require.Equal(t, MetaFrameActionPassthrough, values[0].Action)
		require.Same(t, params.Frame, values[0].Frame)
		require.Equal(t, 0, sfi.numTransfers)
	})
}
