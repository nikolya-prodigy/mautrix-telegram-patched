package upgrades

import (
	"context"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"go.mau.fi/util/dbutil"
)

func newUpgradeTestDB(t *testing.T) *dbutil.Database {
	t.Helper()
	db, err := dbutil.NewWithDialect(":memory:?_foreign_keys=on", "sqlite3")
	require.NoError(t, err)
	db.RawDB.SetMaxOpenConns(1)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	_, err = db.Exec(context.Background(), `
		CREATE TABLE message (id TEXT PRIMARY KEY, room_id TEXT NOT NULL);
		CREATE TABLE reaction (message_id TEXT REFERENCES message(id) ON UPDATE CASCADE);
	`)
	require.NoError(t, err)
	return db
}

func TestFreshSchemaUpgrade(t *testing.T) {
	ctx := context.Background()
	db := newUpgradeTestDB(t)
	require.NoError(t, db.Child("telegram_version", Table, nil).Upgrade(ctx))
	var version int
	require.NoError(t, db.QueryRow(ctx, "SELECT version FROM telegram_version").Scan(&version))
	require.Equal(t, 12, version)
	exists, err := db.TableExists(ctx, "telegram_portal_approval")
	require.NoError(t, err)
	require.True(t, exists)
}

func TestUpstreamSchemaGainsApprovalAndMessageMigration(t *testing.T) {
	ctx := context.Background()
	db := newUpgradeTestDB(t)
	_, err := db.Exec(ctx, `
		CREATE TABLE telegram_version (version INTEGER, compat INTEGER);
		INSERT INTO telegram_version VALUES (9, 2);
	`)
	require.NoError(t, err)
	require.NoError(t, db.Child("telegram_version", Table, nil).Upgrade(ctx))
	exists, err := db.TableExists(ctx, "telegram_portal_approval")
	require.NoError(t, err)
	require.True(t, exists)
}

func TestApprovalSchemaUpgradePreservesData(t *testing.T) {
	ctx := context.Background()
	db := newUpgradeTestDB(t)
	require.NoError(t, db.Child("telegram_version", Table[:11], nil).Upgrade(ctx))
	_, err := db.Exec(ctx, `
		INSERT INTO telegram_portal_approval VALUES
			(7, 42, 'channel:123', '', 'channel', 123, 0, 'Approved', '', 'allowed', 'test', 1, 1);
		INSERT INTO message VALUES
			('41', 'channel:123'), ('123.42', 'channel:123'),
			('43', 'user:456'), ('migrated:44:old', 'channel:123');
		INSERT INTO reaction VALUES ('41');
	`)
	require.NoError(t, err)
	store := db.Child("telegram_version", Table, nil)
	require.NoError(t, store.Upgrade(ctx))
	var migratedID, reactionID, approvalStatus string
	require.NoError(t, db.QueryRow(ctx, "SELECT id FROM message WHERE id LIKE 'migrated:41:%'").Scan(&migratedID))
	require.NoError(t, db.QueryRow(ctx, "SELECT message_id FROM reaction").Scan(&reactionID))
	require.Equal(t, migratedID, reactionID)
	require.NoError(t, db.QueryRow(ctx, "SELECT status FROM telegram_portal_approval WHERE approval_id=7").Scan(&approvalStatus))
	require.Equal(t, "allowed", approvalStatus)
	var untouched int
	require.NoError(t, db.QueryRow(ctx, "SELECT COUNT(*) FROM message WHERE id IN ('123.42', '43', 'migrated:44:old')").Scan(&untouched))
	require.Equal(t, 3, untouched)
	require.NoError(t, store.Upgrade(ctx))
	var afterRerun string
	require.NoError(t, db.QueryRow(ctx, "SELECT id FROM message WHERE id LIKE 'migrated:41:%'").Scan(&afterRerun))
	require.Equal(t, migratedID, afterRerun)
}
