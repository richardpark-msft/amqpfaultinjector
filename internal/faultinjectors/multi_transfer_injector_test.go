package faultinjectors_test

import (
	"context"
	"testing"

	faultinjectors "github.com/richardpark-msft/amqpfaultinjector/internal/faultinjectors"
	"github.com/richardpark-msft/amqpfaultinjector/internal/proto"
	"github.com/richardpark-msft/amqpfaultinjector/internal/proto/frames"
	"github.com/stretchr/testify/require"
)

func TestMultiTransferInjector(t *testing.T) {
	// Initialize the MultiTransferInjector
	injector := faultinjectors.NewMultiTransferInjector()
	var stateMap = &proto.StateMap{}
	// Pass incoming and outgoing ATTACH frames to populate the state map

	// Create a PerformTransfer frame
	var transferBody = &frames.PerformTransfer{
		Payload: []byte("Hello world!"),
		More:    false,
		Handle:  1,
	}
	var transferFrame = &frames.Frame{
		Body: transferBody,
		Header: frames.Header{
			Channel: 0,
		},
	}
	params := faultinjectors.MirrorCallbackParams{
		StateMap: stateMap,
		Frame:    transferFrame,
		Out:      false,
	}
	resultFrames, err := injector.Callback(context.Background(), params)
	require.NoError(t, err)

	// 	Validate the result
	require.Len(t, resultFrames, len(transferBody.Payload)) // Each byte should result in a separate frame

	// now, take all the individual frames and combine them, and unmarshal it.
	var fullPayload []byte

	for i, frame := range resultFrames {
		require.Equal(t, faultinjectors.MetaFrameActionAdded, frame.Action)
		require.NotNil(t, frame.Frame)

		// Validate the payload of each frame
		splitTransferFrame, ok := frame.Frame.Body.(*frames.PerformTransfer)
		require.True(t, ok)
		require.Equal(t, []byte{transferBody.Payload[i]}, splitTransferFrame.Payload)

		// Validate the "More" flag
		expectedMore := i != len(transferBody.Payload)-1
		require.Equal(t, expectedMore, splitTransferFrame.More)

		fullPayload = append(fullPayload, splitTransferFrame.Payload...)
	}

	require.Equal(t, "Hello world!", string(fullPayload))
}
