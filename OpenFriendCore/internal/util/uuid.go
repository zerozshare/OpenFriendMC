/*
 * OpenFriend — Minecraft Java Edition Friends List bridge.
 * Copyright (c) 2026 ZSHARE (https://zpw.jp). Licensed under the MIT License.
 *
 * "Minecraft", "Xbox", "Xbox Live", "Microsoft", and "Mojang" are trademarks
 * of their respective owners. OpenFriend is not affiliated with, endorsed by,
 * sponsored by, or otherwise officially connected to Microsoft Corporation,
 * Mojang AB, or the Xbox brand. See LICENSE for the full notice.
 */
package util

import (
	"fmt"

	"github.com/google/uuid"
)

func ParseUndashed(s string) (uuid.UUID, error) {
	if len(s) == 32 {
		return uuid.Parse(fmt.Sprintf("%s-%s-%s-%s-%s",
			s[0:8], s[8:12], s[12:16], s[16:20], s[20:32]))
	}
	return uuid.Parse(s)
}

func Undashed(u uuid.UUID) string {
	s := u.String()
	out := make([]byte, 0, 32)
	for i := 0; i < len(s); i++ {
		if s[i] != '-' {
			out = append(out, s[i])
		}
	}
	return string(out)
}
