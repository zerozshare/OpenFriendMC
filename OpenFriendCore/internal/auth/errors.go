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

import "fmt"

type Error struct {
	Stage  string
	Status int
	Body   string
	Cause  error
}

func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s", e.Stage, e.Cause.Error())
	}
	return fmt.Sprintf("%s: status=%d body=%s", e.Stage, e.Status, briefBody(e.Body, 200))
}

func (e *Error) Unwrap() error { return e.Cause }

func briefBody(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
