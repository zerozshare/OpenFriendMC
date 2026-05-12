/*
 * OpenFriend — Minecraft Java Edition Friends List bridge.
 * Copyright (c) 2026 ZSHARE (https://zpw.jp). Licensed under the MIT License.
 *
 * "Minecraft", "Xbox", "Xbox Live", "Microsoft", and "Mojang" are trademarks
 * of their respective owners. OpenFriend is not affiliated with, endorsed by,
 * sponsored by, or otherwise officially connected to Microsoft Corporation,
 * Mojang AB, or the Xbox brand. See LICENSE for the full notice.
 */
package auth

import (
	"time"

	"github.com/google/uuid"
)

type Tokens struct {
	ProfileID         uuid.UUID `json:"profileId"`
	Name              string    `json:"name"`
	MojangAccessToken string    `json:"mojangAccessToken"`
	MojangExpiresAt   time.Time `json:"mojangExpiresAt"`
	MsRefreshToken    string    `json:"msRefreshToken"`
}

type MsToken struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
}

type DeviceCodePrompt struct {
	UserCode        string
	VerificationURI string
	Message         string
	DeviceCode      string
	ExpiresIn       time.Duration
	Interval        time.Duration
}
