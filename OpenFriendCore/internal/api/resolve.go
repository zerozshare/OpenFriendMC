/*
 * OpenFriend — Minecraft Java Edition Friends List bridge.
 * Copyright (c) 2026 ZSHARE (https://zpw.jp). Licensed under the MIT License.
 *
 * "Minecraft", "Xbox", "Xbox Live", "Microsoft", and "Mojang" are trademarks
 * of their respective owners. OpenFriend is not affiliated with, endorsed by,
 * sponsored by, or otherwise officially connected to Microsoft Corporation,
 * Mojang AB, or the Xbox brand. See LICENSE for the full notice.
 */
package api

import (
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"jp.zpw.openfriend/internal/util"
)

type Peer struct {
	Name      string
	ProfileID uuid.UUID
	PMID      uuid.UUID
}

func ResolvePeer(client *Client, query string) (*Peer, error) {
	if raw, err := util.ParseUndashed(query); err == nil {
		return resolveByPmid(client, raw)
	}

	friends, err := client.GetFriends()
	if err != nil {
		return nil, fmt.Errorf("list friends: %w", err)
	}
	if friends == nil {
		return nil, errors.New("friends list unavailable (304 with no cache)")
	}

	var match *FriendDto
	for i, f := range friends.Friends {
		if strings.EqualFold(f.Name, query) {
			match = &friends.Friends[i]
			break
		}
	}
	if match == nil {
		return nil, fmt.Errorf("no friend named %q (have %d friends)", query, len(friends.Friends))
	}

	pres, err := client.PostPresence(PresenceRequest{
		Status:   StatusOnline,
		JoinInfo: nil,
	})
	if err != nil {
		return nil, fmt.Errorf("fetch presence: %w", err)
	}
	if pres == nil {
		return nil, errors.New("presence response cached; can't resolve pmid without a fresh response")
	}
	wantID := match.ProfileID.UUID()
	for _, p := range pres.Presence {
		if p.ProfileID.UUID() == wantID {
			return &Peer{Name: match.Name, ProfileID: wantID, PMID: p.PMID.UUID()}, nil
		}
	}
	return nil, fmt.Errorf("friend %q is not currently sharing presence; cannot resolve pmid", query)
}

func resolveByPmid(client *Client, pmid uuid.UUID) (*Peer, error) {
	pres, err := client.PostPresence(PresenceRequest{
		Status:   StatusOnline,
		JoinInfo: nil,
	})
	if err != nil {
		return nil, fmt.Errorf("fetch presence: %w", err)
	}
	peer := &Peer{PMID: pmid}
	if pres != nil {
		for _, p := range pres.Presence {
			if p.PMID.UUID() == pmid {
				peer.ProfileID = p.ProfileID.UUID()
				break
			}
		}
	}
	if peer.ProfileID == uuid.Nil {
		return peer, nil
	}
	friends, err := client.GetFriends()
	if err == nil && friends != nil {
		for _, f := range friends.Friends {
			if f.ProfileID.UUID() == peer.ProfileID {
				peer.Name = f.Name
				break
			}
		}
	}
	return peer, nil
}
