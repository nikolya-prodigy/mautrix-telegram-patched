package updates

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"go.mau.fi/mautrix-telegram/pkg/gotd/tg"
)

func TestDifferenceRetryDrainsExternalAndInternalUpdates(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	s := &internalState{externalQueue: make(chan tracedUpdate), internalQueue: make(chan tracedUpdate)}
	waitCtx, stopWaiting := context.WithCancel(ctx)
	defer stopWaiting()
	done := make(chan error, 1)
	go func() { done <- s.waitToRetryDifference(waitCtx, time.Hour) }()
	for i, queue := range []chan tracedUpdate{s.externalQueue, s.internalQueue} {
		select {
		case queue <- tracedUpdate{update: &tg.Updates{Date: i + 1}}:
		case <-ctx.Done():
			t.Fatal("updates blocked while waiting to retry difference")
		}
	}
	stopWaiting()
	select {
	case err := <-done:
		require.ErrorIs(t, err, context.Canceled)
	case <-ctx.Done():
		t.Fatal("difference retry did not stop on cancellation")
	}
	require.Len(t, s.deferred, 2)
	require.Equal(t, 1, s.deferred[0].update.(*tg.Updates).Date)
	require.Equal(t, 2, s.deferred[1].update.(*tg.Updates).Date)
}

func TestPushToChannelDrainsUpdatesWhileChannelIsBlocked(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	s := &internalState{externalQueue: make(chan tracedUpdate), internalQueue: make(chan tracedUpdate)}
	channel := &channelState{runCtx: ctx, updates: make(chan channelUpdate)}
	done := make(chan error, 1)
	go func() { done <- s.pushToChannel(ctx, channel, channelUpdate{}) }()
	for i, queue := range []chan tracedUpdate{s.externalQueue, s.internalQueue} {
		select {
		case queue <- tracedUpdate{update: &tg.Updates{Date: i + 1}}:
		case <-ctx.Done():
			t.Fatal("blocked channel prevented draining updates")
		}
	}
	select {
	case <-channel.updates:
	case <-ctx.Done():
		t.Fatal("update was not delivered after the channel became writable")
	}
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-ctx.Done():
		t.Fatal("push to channel did not finish")
	}
	require.Len(t, s.deferred, 2)
	require.Equal(t, 1, s.deferred[0].update.(*tg.Updates).Date)
	require.Equal(t, 2, s.deferred[1].update.(*tg.Updates).Date)
}
