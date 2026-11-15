package tgtest_test

import (
	"context"
	"crypto/rand"
	"io"
	"testing"
	"time"

	"github.com/go-faster/errors"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"go.mau.fi/mautrix-telegram/pkg/gotd/crypto"
	"go.mau.fi/mautrix-telegram/pkg/gotd/session"
	"go.mau.fi/mautrix-telegram/pkg/gotd/tdsync"
	"go.mau.fi/mautrix-telegram/pkg/gotd/telegram"
	"go.mau.fi/mautrix-telegram/pkg/gotd/tg"
	"go.mau.fi/mautrix-telegram/pkg/gotd/tgtest/cluster"
	"go.mau.fi/mautrix-telegram/pkg/gotd/tgtest/services"
	"go.mau.fi/mautrix-telegram/pkg/gotd/tgtest/services/config"
)

func TestSessionHandle(t *testing.T) {
	test := func(storage session.Storage, timeout time.Duration, t *testing.T) error {
		log := zaptest.NewLogger(t)
		defer func() { _ = log.Sync() }()

		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		g := tdsync.NewCancellableGroup(ctx)
		c := cluster.NewCluster(cluster.Options{
			Logger: log.Named("cluster"),
		})
		d := c.Dispatch(2, "server").Fallback(services.NotImplemented)
		config.NewService(&tg.Config{}, &tg.CDNConfig{}).Register(d)

		g.Go(c.Up)

		g.Go(func(ctx context.Context) error {
			select {
			case <-c.Ready():
			case <-ctx.Done():
				return ctx.Err()
			}
			defer g.Cancel()

			client := telegram.NewClient(telegram.TestAppID, telegram.TestAppHash, telegram.Options{
				PublicKeys:     c.Keys(),
				DC:             2,
				DCList:         c.List(),
				Resolver:       c.Resolver(),
				NoUpdates:      true,
				Logger:         log.Named("client"),
				SessionStorage: storage,
				RetryInterval:  100 * time.Millisecond,
			})

			return client.Run(ctx, func(ctx context.Context) error {
				return nil
			})
		})

		if err := g.Wait(); err != nil && !errors.Is(err, context.Canceled) {
			return errors.Wrap(err, "wait")
		}
		return nil
	}

	t.Run("Empty", func(t *testing.T) {
		a := require.New(t)
		storage := session.StorageMemory{}
		a.NoError(test(&storage, time.Minute, t))

		_, err := storage.Bytes(nil)
		a.NoError(err, "Must create new session")
	})
	t.Run("Unknown", func(t *testing.T) {
		a := require.New(t)
		loader := session.Loader{Storage: &session.StorageMemory{}}
		ctx := context.Background()

		key := crypto.Key{}
		_, err := io.ReadFull(rand.Reader, key[:])
		a.NoError(err)
		authKey := key.WithID()

		was := &session.Data{
			DC:        2,
			AuthKey:   authKey.Value[:],
			AuthKeyID: authKey.ID[:],
		}
		a.NoError(loader.Save(context.Background(), was))
		// Allow the one-minute reconnect grace period before the key is replaced.
		a.NoError(test(loader.Storage, 2*time.Minute, t))

		data, err := loader.Load(ctx)
		a.NoError(err)
		a.NotEqual(was, data, "Must regenerate session")
	})
}
