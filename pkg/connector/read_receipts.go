// mautrix-telegram - A Matrix-Telegram puppeting bridge.
// Copyright (C) 2026 Tulir Asokan
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package connector

import (
	"context"
	"strings"

	"github.com/rs/zerolog"
	"maunium.net/go/mautrix/bridgev2"
	"maunium.net/go/mautrix/bridgev2/database"
	"maunium.net/go/mautrix/bridgev2/networkid"
	"maunium.net/go/mautrix/bridgev2/simplevent"

	"go.mau.fi/mautrix-telegram/pkg/connector/ids"
)

const readReceiptFallbackMessageLimit = 1000

func isUsableReadReceiptTarget(message *database.Message) bool {
	return message != nil && message.MXID != "" && !message.HasFakeMXID() &&
		!strings.HasPrefix(message.MXID.String(), database.TxnMXIDPrefix)
}

func findReadReceiptFallback(messages []*database.Message, maxID int) networkid.MessageID {
	var fallback networkid.MessageID
	fallbackMessageID := -1
	for _, message := range messages {
		if !isUsableReadReceiptTarget(message) {
			continue
		}
		_, messageID, err := ids.ParseMessageID(message.ID)
		if err != nil || messageID >= maxID || messageID <= fallbackMessageID {
			continue
		}
		fallback = message.ID
		fallbackMessageID = messageID
	}
	return fallback
}

func (tc *TelegramClient) makeOwnReadReceipt(
	ctx context.Context,
	portalKey networkid.PortalKey,
	maxID int,
	logContext func(zerolog.Context) zerolog.Context,
) (*simplevent.Receipt, error) {
	if maxID <= 0 {
		return nil, nil
	}

	targetID := ids.MakeMessageID(portalKey, maxID)
	target, err := tc.main.Bridge.DB.Message.GetLastPartByID(ctx, portalKey.Receiver, targetID)
	if err != nil {
		return nil, err
	}

	var fallbackID networkid.MessageID
	if !isUsableReadReceiptTarget(target) {
		messages, err := tc.main.Bridge.DB.Message.GetLastNInPortal(ctx, portalKey, readReceiptFallbackMessageLimit)
		if err != nil {
			return nil, err
		}
		fallbackID = findReadReceiptFallback(messages, maxID)
	}

	receipt := &simplevent.Receipt{
		EventMeta: simplevent.EventMeta{
			Type:      bridgev2.RemoteEventReadReceipt,
			PortalKey: portalKey,
			Sender:    tc.mySender(),
			LogContext: func(c zerolog.Context) zerolog.Context {
				if logContext != nil {
					c = logContext(c)
				}
				c = c.Int("read_max_id", maxID)
				if fallbackID != "" {
					c = c.Str("read_fallback_target", string(fallbackID))
				}
				return c
			},
		},
		LastTarget:          targetID,
		ReadUpToStreamOrder: int64(maxID),
	}
	if fallbackID != "" {
		receipt.Targets = []networkid.MessageID{fallbackID}
	}
	return receipt, nil
}
