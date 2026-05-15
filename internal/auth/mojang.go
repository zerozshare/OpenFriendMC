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
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"jp.zpw.openfriend/internal/util"
)

func ExchangeForMojang(ms *MsToken) (*Tokens, error) {
	xbl, err := xboxLive(ms.AccessToken)
	if err != nil {
		return nil, err
	}
	x, err := xsts(xbl.Token)
	if err != nil {
		return nil, err
	}

	body := map[string]any{
		"identityToken": "XBL3.0 x=" + x.UserHash + ";" + x.Token,
	}
	buf, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", mojangLogin, bytes.NewReader(buf))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := util.Client.Do(req)
	if err != nil {
		return nil, &Error{Stage: "mojang-login", Cause: err}
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, &Error{Stage: "mojang-login", Status: resp.StatusCode, Body: string(raw)}
	}
	var j map[string]any
	if err := json.Unmarshal(raw, &j); err != nil {
		return nil, &Error{Stage: "mojang-login", Cause: err}
	}
	mojangToken, _ := j["access_token"].(string)
	expiresIn, _ := j["expires_in"].(float64)

	prof, err := fetchProfile(mojangToken)
	if err != nil {
		return nil, err
	}

	return &Tokens{
		ProfileID:         prof.id,
		Name:              prof.name,
		MojangAccessToken: mojangToken,
		MojangExpiresAt:   time.Now().Add(time.Duration(expiresIn) * time.Second),
		MsRefreshToken:    ms.RefreshToken,
	}, nil
}

func SilentRefresh(refreshToken string) (*Tokens, error) {
	ms, err := RefreshMsToken(refreshToken)
	if err != nil {
		return nil, err
	}
	return ExchangeForMojang(ms)
}

type profile struct {
	id   uuid.UUID
	name string
}

func fetchProfile(mojangToken string) (*profile, error) {
	req, _ := http.NewRequest("GET", mcProfileURL, nil)
	req.Header.Set("Authorization", "Bearer "+mojangToken)
	resp, err := util.Client.Do(req)
	if err != nil {
		return nil, &Error{Stage: "mc-profile", Cause: err}
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, &Error{Stage: "mc-profile", Status: resp.StatusCode, Body: string(raw)}
	}
	var j map[string]any
	if err := json.Unmarshal(raw, &j); err != nil {
		return nil, &Error{Stage: "mc-profile", Cause: err}
	}
	idStr, _ := j["id"].(string)
	name, _ := j["name"].(string)
	id, err := util.ParseUndashed(idStr)
	if err != nil {
		return nil, &Error{Stage: "mc-profile", Cause: err}
	}
	return &profile{id: id, name: name}, nil
}
