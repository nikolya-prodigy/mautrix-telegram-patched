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

	"github.com/rs/zerolog"
	"maunium.net/go/mautrix/bridgev2"
	"maunium.net/go/mautrix/bridgev2/networkid"
)

const (
	createReasonApprovalCommand        = "approval_command"
	createReasonAvailableReactions     = "available_reactions"
	createReasonBackgroundNotification = "background_notification"
	createReasonBackgroundProfile      = "background_profile_update"
	createReasonChatInfo               = "chat_info"
	createReasonDialogSync             = "dialog_sync"
	createReasonIncomingMessage        = "incoming_message"
	createReasonMatrixAction           = "matrix_action"
	createReasonMessageConversion      = "message_conversion"
	createReasonPhoneCall              = "phone_call"
	createReasonPushNotification       = "push_notification"
	createReasonResolveIdentifier      = "resolve_identifier"
	createReasonServiceMessage         = "service_message"
	createReasonUpdateChannel          = "update_channel"
)

func (tc *TelegramClient) getGhostByIDWithPolicy(ctx context.Context, ghostID networkid.UserID, reason string, createIfMissing bool) (*bridgev2.Ghost, error) {
	if ghostID == "" {
		return nil, nil
	}
	log := zerolog.Ctx(ctx)
	ghost, err := tc.main.Bridge.GetExistingGhostByID(ctx, ghostID)
	if err != nil {
		return nil, err
	} else if ghost != nil || !createIfMissing {
		if ghost == nil {
			log.Debug().
				Str("creation_kind", "ghost").
				Str("creation_reason", reason).
				Str("ghost_id", string(ghostID)).
				Msg("Denied Matrix ghost creation")
		}
		return ghost, nil
	}
	log.Info().
		Str("creation_kind", "ghost").
		Str("creation_reason", reason).
		Str("ghost_id", string(ghostID)).
		Msg("Allowing Matrix ghost creation")
	return tc.main.Bridge.GetGhostByID(ctx, ghostID)
}

func (tc *TelegramClient) getPortalByKeyWithPolicy(ctx context.Context, portalKey networkid.PortalKey, reason string, createIfMissing bool) (*bridgev2.Portal, error) {
	log := zerolog.Ctx(ctx)
	portal, err := tc.main.Bridge.GetExistingPortalByKey(ctx, portalKey)
	if err != nil {
		return nil, err
	} else if portal != nil || !createIfMissing {
		if portal == nil {
			log.Debug().
				Str("creation_kind", "portal").
				Str("creation_reason", reason).
				Stringer("portal_key", portalKey).
				Msg("Denied Matrix portal creation")
		}
		return portal, nil
	}
	log.Info().
		Str("creation_kind", "portal").
		Str("creation_reason", reason).
		Stringer("portal_key", portalKey).
		Msg("Allowing Matrix portal creation")
	return tc.main.Bridge.GetPortalByKey(ctx, portalKey)
}
