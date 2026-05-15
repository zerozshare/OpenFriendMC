/*
 * OpenFriend — Minecraft Java Edition Friends List bridge.
 * Copyright (c) 2026 ZSHARE (https://zpw.jp). Licensed under the MIT License.
 *
 * "Minecraft", "Xbox", "Xbox Live", "Microsoft", and "Mojang" are trademarks
 * of their respective owners. OpenFriend is not affiliated with, endorsed by,
 * sponsored by, or otherwise officially connected to Microsoft Corporation,
 * Mojang AB, or the Xbox brand. See LICENSE for the full notice.
 */
package signaling

import (
	"context"
	"encoding/json"
	"errors"
)

type TurnAuth struct {
	Username string
	Password string
	URLs     []string
}

func (c *Client) RequestTurnAuth(ctx context.Context) (*TurnAuth, error) {
	raw, err := c.sendRequestAwait(ctx, "Signaling_TurnAuth_v1_0", []any{})
	if err != nil {
		return nil, err
	}
	var obj struct {
		TurnAuthServers []struct {
			Username string   `json:"Username"`
			Password string   `json:"Password"`
			Urls     []string `json:"Urls"`
		} `json:"TurnAuthServers"`
	}
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil, err
	}
	if len(obj.TurnAuthServers) == 0 {
		return nil, errors.New("turn auth returned no servers")
	}
	out := &TurnAuth{
		Username: obj.TurnAuthServers[0].Username,
		Password: obj.TurnAuthServers[0].Password,
	}
	for _, s := range obj.TurnAuthServers {
		out.URLs = append(out.URLs, s.Urls...)
	}
	return out, nil
}
