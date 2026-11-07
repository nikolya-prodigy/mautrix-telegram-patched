package store

import (
	"context"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"go.mau.fi/util/dbutil"

	"go.mau.fi/mautrix-telegram/pkg/gotd/telegram/updates"
)

func TestStateSettersUpdateOnlyScopedUser(t *testing.T) {
	ctx := context.Background()
	db, err := dbutil.NewWithDialect(":memory:", "sqlite3")
	require.NoError(t, err)
	db.RawDB.SetMaxOpenConns(1)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	_, err = db.Exec(ctx, `CREATE TABLE telegram_user_state (
		user_id BIGINT PRIMARY KEY, pts BIGINT, qts BIGINT, date BIGINT, seq BIGINT
	)`)
	require.NoError(t, err)
	s := &ScopedStore{db: db, telegramUserID: 42}
	other := &ScopedStore{db: db, telegramUserID: 7}
	original := updates.State{Pts: 10, Qts: 20, Date: 30, Seq: 40}
	require.NoError(t, other.SetState(ctx, 7, original))
	for _, test := range []struct {
		name string
		set  func() error
		want updates.State
	}{
		{"pts", func() error { return s.SetPts(ctx, 42, 100) }, updates.State{Pts: 100, Qts: 20, Date: 30, Seq: 40}},
		{"qts", func() error { return s.SetQts(ctx, 42, 200) }, updates.State{Pts: 10, Qts: 200, Date: 30, Seq: 40}},
		{"date", func() error { return s.SetDate(ctx, 42, 300) }, updates.State{Pts: 10, Qts: 20, Date: 300, Seq: 40}},
		{"seq", func() error { return s.SetSeq(ctx, 42, 400) }, updates.State{Pts: 10, Qts: 20, Date: 30, Seq: 400}},
		{"date_seq", func() error { return s.SetDateSeq(ctx, 42, 300, 400) }, updates.State{Pts: 10, Qts: 20, Date: 300, Seq: 400}},
	} {
		t.Run(test.name, func(t *testing.T) {
			require.NoError(t, s.SetState(ctx, 42, original))
			require.NoError(t, test.set())
			got, found, err := s.GetState(ctx, 42)
			require.NoError(t, err)
			require.True(t, found)
			require.Equal(t, test.want, got)
			gotOther, found, err := other.GetState(ctx, 7)
			require.NoError(t, err)
			require.True(t, found)
			require.Equal(t, original, gotOther)
		})
	}
}
