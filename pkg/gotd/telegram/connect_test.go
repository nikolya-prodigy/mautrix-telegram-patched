package telegram

import (
	"context"
	"testing"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/gotd/neo"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"go.mau.fi/mautrix-telegram/pkg/gotd/bin"
	"go.mau.fi/mautrix-telegram/pkg/gotd/crypto"
	"go.mau.fi/mautrix-telegram/pkg/gotd/exchange"
	"go.mau.fi/mautrix-telegram/pkg/gotd/pool"
	"go.mau.fi/mautrix-telegram/pkg/gotd/proto/codec"
)

type fingerprintNotFoundConn struct{}

func (m fingerprintNotFoundConn) Run(context.Context) error {
	return exchange.ErrKeyFingerprintNotFound
}

func TestAuthKeyNotFoundGracePeriod(t *testing.T) {
	clock := neo.NewTime(time.Now())
	var key crypto.Key
	key[0] = 1
	initial := pool.Session{DC: 2, AuthKey: key.WithID(), Salt: 123}
	client := &Client{clock: clock, log: zap.NewNop(), session: pool.NewSyncSession(initial)}
	missing := &codec.ProtocolErr{Code: codec.CodeAuthKeyNotFound}
	client.updateAuthKeyNotFound(missing)
	require.Equal(t, initial, client.session.Load())
	clock.Set(clock.Now().Add(authKeyNotFoundTimeout))
	client.updateAuthKeyNotFound(missing)
	require.Equal(t, initial, client.session.Load(), "keep the key throughout the grace period")
	clock.Set(clock.Now().Add(time.Second))
	client.updateAuthKeyNotFound(missing)
	require.Equal(t, pool.Session{DC: 2}, client.session.Load())
	require.True(t, client.authKeyNotFoundSince.IsZero())
}

func TestOtherReconnectErrorResetsAuthKeyNotFoundTimer(t *testing.T) {
	clock := neo.NewTime(time.Now())
	var key crypto.Key
	key[0] = 1
	initial := pool.Session{DC: 2, AuthKey: key.WithID()}
	client := &Client{clock: clock, log: zap.NewNop(), session: pool.NewSyncSession(initial)}
	missing := &codec.ProtocolErr{Code: codec.CodeAuthKeyNotFound}
	client.updateAuthKeyNotFound(missing)
	clock.Set(clock.Now().Add(authKeyNotFoundTimeout + time.Second))
	client.updateAuthKeyNotFound(context.DeadlineExceeded)
	require.True(t, client.authKeyNotFoundSince.IsZero())
	client.updateAuthKeyNotFound(missing)
	require.Equal(t, initial, client.session.Load(), "an unrelated failure must break the consecutive auth-key error period")
}

func (m fingerprintNotFoundConn) Invoke(context.Context, bin.Encoder, bin.Decoder) error {
	return nil
}

func (m fingerprintNotFoundConn) Ping(context.Context) error {
	return nil
}

func (m fingerprintNotFoundConn) Ready() <-chan struct{} {
	return nil
}

func TestClient_reconnectUntilClosed(t *testing.T) {
	client := Client{
		newConnBackoff: func() backoff.BackOff {
			return backoff.NewConstantBackOff(time.Nanosecond)
		},
		log: zap.NewNop(),
	}
	client.init()
	client.conn = fingerprintNotFoundConn{}

	ctx := context.Background()
	require.ErrorIs(t, client.reconnectUntilClosed(ctx), exchange.ErrKeyFingerprintNotFound)
}
