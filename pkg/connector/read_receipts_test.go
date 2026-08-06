// mautrix-telegram - A Matrix-Telegram puppeting bridge.
// Copyright (C) 2026 Tulir Asokan
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package connector

import (
	"testing"

	"maunium.net/go/mautrix/bridgev2/database"
	"maunium.net/go/mautrix/bridgev2/networkid"
	"maunium.net/go/mautrix/id"
)

func TestFindReadReceiptFallback(t *testing.T) {
	message := func(messageID, mxid string) *database.Message {
		return &database.Message{ID: networkid.MessageID(messageID), MXID: id.EventID(mxid)}
	}
	tests := []struct {
		name     string
		messages []*database.Message
		maxID    int
		want     networkid.MessageID
	}{
		{
			name: "nearest earlier message",
			messages: []*database.Message{
				message("100", "$100"),
				message("104", "$104"),
				message("108", "$108"),
			},
			maxID: 105,
			want:  "104",
		},
		{
			name: "channel message IDs",
			messages: []*database.Message{
				message("1234.40", "$40"),
				message("1234.42", "$42"),
			},
			maxID: 43,
			want:  "1234.42",
		},
		{
			name: "exact message is reserved for primary target",
			messages: []*database.Message{
				message("99", "$99"),
				message("100", "$100"),
			},
			maxID: 100,
			want:  "99",
		},
		{
			name: "non-event and invalid messages are ignored",
			messages: []*database.Message{
				message("bad", "$bad"),
				message("98", "$98"),
				message("99", database.FakeMXIDPrefix+"placeholder"),
				message("97", database.TxnMXIDPrefix+"pending"),
				message("96", ""),
			},
			maxID: 100,
			want:  "98",
		},
		{
			name: "no earlier message",
			messages: []*database.Message{
				message("100", "$100"),
			},
			maxID: 100,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := findReadReceiptFallback(test.messages, test.maxID); got != test.want {
				t.Fatalf("findReadReceiptFallback() = %q, want %q", got, test.want)
			}
		})
	}
}
