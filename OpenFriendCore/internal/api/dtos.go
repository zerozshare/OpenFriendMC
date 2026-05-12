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
	"time"

	"github.com/google/uuid"
)

type PresenceStatus string

const (
	StatusOnline              PresenceStatus = "ONLINE"
	StatusPlayingOffline      PresenceStatus = "PLAYING_OFFLINE"
	StatusPlayingRealms       PresenceStatus = "PLAYING_REALMS"
	StatusPlayingServer       PresenceStatus = "PLAYING_SERVER"
	StatusPlayingHostedServer PresenceStatus = "PLAYING_HOSTED_SERVER"
	StatusOffline             PresenceStatus = "OFFLINE"
)

type FriendDto struct {
	ProfileID undashedUUID `json:"profileId"`
	Name      string       `json:"name"`
}

type FriendsListResponse struct {
	Friends          []FriendDto `json:"friends"`
	IncomingRequests []FriendDto `json:"incomingRequests"`
	OutgoingRequests []FriendDto `json:"outgoingRequests"`
}

type JoinInfoUpdate struct {
	Value   *string        `json:"value"`
	Invites []undashedUUID `json:"invites"`
}

type JoinInfoDto struct {
	Value   *string `json:"value"`
	Invited bool    `json:"invited"`
}

type PresenceRequest struct {
	Status   PresenceStatus  `json:"status"`
	JoinInfo *JoinInfoUpdate `json:"joinInfo,omitempty"`
}

type PresenceStatusDto struct {
	ProfileID   undashedUUID   `json:"profileId"`
	PMID        undashedUUID   `json:"pmid"`
	Status      PresenceStatus `json:"status"`
	JoinInfo    *JoinInfoDto   `json:"joinInfo"`
	LastUpdated time.Time      `json:"lastUpdated"`
}

type PresenceResponse struct {
	Presence []PresenceStatusDto `json:"presence"`
}

type FriendActionRequest struct {
	Name       *string       `json:"name,omitempty"`
	ProfileID  *undashedUUID `json:"profileId,omitempty"`
	UpdateType string        `json:"updateType"`
}

func AddByName(name string) FriendActionRequest {
	return FriendActionRequest{Name: &name, UpdateType: "ADD"}
}

func AddByID(id uuid.UUID) FriendActionRequest {
	u := undashedUUID(id)
	return FriendActionRequest{ProfileID: &u, UpdateType: "ADD"}
}

func RemoveByID(id uuid.UUID) FriendActionRequest {
	u := undashedUUID(id)
	return FriendActionRequest{ProfileID: &u, UpdateType: "REMOVE"}
}

func NewJoinInfoUpdate(invites []uuid.UUID) *JoinInfoUpdate {
	inv := make([]undashedUUID, len(invites))
	for i, u := range invites {
		inv[i] = undashedUUID(u)
	}
	return &JoinInfoUpdate{Value: nil, Invites: inv}
}
