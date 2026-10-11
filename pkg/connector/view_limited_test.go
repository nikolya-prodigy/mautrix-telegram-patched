// mautrix-telegram - A Matrix-Telegram puppeting bridge.
// Copyright (C) 2026 Nikolya Prodigy
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

package connector

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.mau.fi/util/jsontime"
	"maunium.net/go/mautrix/bridgev2"
	"maunium.net/go/mautrix/bridgev2/database"
	"maunium.net/go/mautrix/event"

	"go.mau.fi/mautrix-telegram/pkg/gotd/tg"
)

func TestDisabledViewOnceMediaDoesNotUpload(t *testing.T) {
	photo := &tg.MessageMediaPhoto{}
	photo.SetTTLSeconds(telegramViewOnceTTL)
	document := &tg.MessageMediaDocument{}
	document.SetTTLSeconds(telegramViewOnceTTL)
	for _, test := range []struct {
		name  string
		media tg.MessageMediaClass
	}{
		{"photo", photo},
		{"file", document},
	} {
		t.Run(test.name, func(t *testing.T) {
			tc := &TelegramClient{main: &TelegramConnector{Config: TelegramConfig{DisableViewOnce: true}}}
			portal := &bridgev2.Portal{Portal: &database.Portal{}}
			// A nil Telegram client ensures the disabled media never reaches upload setup.
			converted, disappearing := tc.convertMediaRequiringUpload(context.Background(), portal, nil, &tg.Message{ID: 1}, test.media, false)
			require.Nil(t, disappearing)
			require.Equal(t, event.MsgNotice, converted.Content.MsgType)
			require.Contains(t, converted.Content.Body, "view once "+test.name)
			require.Nil(t, converted.Content.BeeperViewLimited)
		})
	}
}

func TestTelegramMediaTTL(t *testing.T) {
	for _, seconds := range []int{1, 3, 10, 30, 60, telegramViewOnceTTL} {
		ttl, err := telegramMediaTTL(telegramViewLimit(seconds))
		require.NoError(t, err)
		require.Equal(t, seconds, ttl)
	}
	for _, limit := range []*event.BeeperViewLimitedMedia{
		{Type: "count", Count: 2},
		{Type: "count", Count: 1, Time: jsontime.MS(time.Second)},
		{Type: "time", Time: jsontime.MS(1500 * time.Millisecond)},
		{Type: "time", Time: jsontime.MS(61 * time.Second)},
	} {
		_, err := telegramMediaTTL(limit)
		require.ErrorIs(t, err, bridgev2.ErrUnsupportedViewLimitedType)
	}
}
