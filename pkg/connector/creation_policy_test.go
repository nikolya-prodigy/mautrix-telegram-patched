// mautrix-telegram - A Matrix-Telegram puppeting bridge.
// Copyright (C) 2026 Tulir Asokan
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package connector

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"go.mau.fi/mautrix-telegram/pkg/connector/ids"
)

func TestMatrixCreationCallsGoThroughPolicy(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to locate test file")
	}
	connectorDir := filepath.Dir(filename)
	directCreateCall := regexp.MustCompile(`\.Get(?:GhostByID|PortalByKey)\s*\(`)

	var offenders []string
	err := filepath.WalkDir(connectorDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		} else if d.IsDir() {
			if d.Name() == "store" {
				return filepath.SkipDir
			}
			return nil
		}
		base := filepath.Base(path)
		if !strings.HasSuffix(base, ".go") || strings.HasSuffix(base, "_test.go") || base == "creation_policy.go" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for lineNo, line := range strings.Split(string(data), "\n") {
			if directCreateCall.MatchString(line) {
				rel, _ := filepath.Rel(connectorDir, path)
				offenders = append(offenders, rel+":"+strconv.Itoa(lineNo+1)+": "+strings.TrimSpace(line))
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(offenders) > 0 {
		t.Fatalf("direct Matrix ghost/portal creation calls must use creation_policy.go:\n%s", strings.Join(offenders, "\n"))
	}
}

func TestPortalApprovalAutoCreateMatrix(t *testing.T) {
	botsAllowed := true
	botsDenied := false
	tests := []struct {
		name string
		cfg  func(*TelegramConfig)
		info portalApprovalInfo
		want bool
	}{
		{
			name: "private chats can be auto-created",
			cfg: func(cfg *TelegramConfig) {
				cfg.PortalApproval.AutoCreate.PrivateChats = true
			},
			info: portalApprovalInfo{PeerType: string(ids.PeerTypeUser)},
			want: true,
		},
		{
			name: "bots can be denied separately from private chats",
			cfg: func(cfg *TelegramConfig) {
				cfg.PortalApproval.AutoCreate.PrivateChats = true
				cfg.PortalApproval.AutoCreate.Bots = &botsDenied
			},
			info: portalApprovalInfo{PeerType: string(ids.PeerTypeUser), IsBot: true},
			want: false,
		},
		{
			name: "bots can be allowed separately from private chats",
			cfg: func(cfg *TelegramConfig) {
				cfg.PortalApproval.AutoCreate.PrivateChats = false
				cfg.PortalApproval.AutoCreate.Bots = &botsAllowed
			},
			info: portalApprovalInfo{PeerType: string(ids.PeerTypeUser), IsBot: true},
			want: true,
		},
		{
			name: "groups are denied by default",
			cfg:  func(cfg *TelegramConfig) {},
			info: portalApprovalInfo{PeerType: string(ids.PeerTypeChat)},
			want: false,
		},
		{
			name: "supergroups use their own setting",
			cfg: func(cfg *TelegramConfig) {
				cfg.PortalApproval.AutoCreate.Supergroups = true
			},
			info: portalApprovalInfo{PeerType: "supergroup"},
			want: true,
		},
		{
			name: "channels are denied by default",
			cfg:  func(cfg *TelegramConfig) {},
			info: portalApprovalInfo{PeerType: string(ids.PeerTypeChannel)},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := TelegramConfig{}
			tt.cfg(&cfg)
			client := &TelegramClient{main: &TelegramConnector{Config: cfg}}
			if got := client.portalApprovalAutoAllowed(tt.info); got != tt.want {
				t.Fatalf("portalApprovalAutoAllowed() = %v, want %v", got, tt.want)
			}
		})
	}
}
