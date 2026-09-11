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
	"testing"

	"github.com/stretchr/testify/require"
	"maunium.net/go/mautrix/bridgev2"
	"maunium.net/go/mautrix/bridgev2/status"
)

func TestShouldEscalateDeadConnection(t *testing.T) {
	bridgeState := &bridgev2.BridgeStateQueue{}
	tc := &TelegramClient{
		userLogin: &bridgev2.UserLogin{BridgeState: bridgeState},
	}
	connectionState := tc.connectionState.Add(1)

	bridgeState.SetPrev(status.BridgeState{StateEvent: status.StateTransientDisconnect})
	require.True(t, tc.shouldEscalateDeadConnection(connectionState))

	tc.connectionState.Add(1)
	require.False(t, tc.shouldEscalateDeadConnection(connectionState))

	connectionState = tc.connectionState.Load()
	bridgeState.SetPrev(status.BridgeState{StateEvent: status.StateConnected})
	require.False(t, tc.shouldEscalateDeadConnection(connectionState))
}
