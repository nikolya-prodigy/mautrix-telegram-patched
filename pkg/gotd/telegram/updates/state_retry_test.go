package updates

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"go.mau.fi/mautrix-telegram/pkg/gotd/tg"
)

func TestDifferenceRetryDrainsExternalUpdates(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	s := &internalState{externalQueue: make(chan tracedUpdate)}
	waitCtx, stopWaiting := context.WithCancel(ctx)
	done := make(chan error, 1)
	go func() { done <- s.waitToRetryDifference(waitCtx, time.Hour) }()
	for _, date := range []int{1, 2} {
		select {
		case s.externalQueue <- tracedUpdate{update: &tg.Updates{Date: date}}:
		case <-ctx.Done():
			t.Fatal("external updates blocked while waiting to retry difference")
		}
	}
	stopWaiting()
	select {
	case err := <-done:
		require.ErrorIs(t, err, context.Canceled)
	case <-ctx.Done():
		t.Fatal("difference retry did not stop on cancellation")
	}
	require.Len(t, s.deferredExternal, 2)
	require.Equal(t, 1, s.deferredExternal[0].update.(*tg.Updates).Date)
	require.Equal(t, 2, s.deferredExternal[1].update.(*tg.Updates).Date)
}
