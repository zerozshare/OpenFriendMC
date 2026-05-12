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
	"github.com/google/uuid"
	"jp.zpw.openfriend/internal/util"
)

type undashedUUID uuid.UUID

func (u undashedUUID) MarshalJSON() ([]byte, error) {
	return []byte(`"` + util.Undashed(uuid.UUID(u)) + `"`), nil
}

func (u *undashedUUID) UnmarshalJSON(b []byte) error {
	if len(b) < 2 {
		return nil
	}
	s := string(b[1 : len(b)-1])
	parsed, err := util.ParseUndashed(s)
	if err != nil {
		return err
	}
	*u = undashedUUID(parsed)
	return nil
}

func (u undashedUUID) UUID() uuid.UUID { return uuid.UUID(u) }
