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
	"errors"
	"fmt"
	"io"
	"net/http"

	"jp.zpw.openfriend/internal/util"
)

type xblAuth struct {
	Token    string
	UserHash string
}

func xboxLive(msAccess string) (*xblAuth, error) {
	body := map[string]any{
		"Properties": map[string]any{
			"AuthMethod": "RPS",
			"SiteName":   "user.auth.xboxlive.com",
			"RpsTicket":  msAccess,
		},
		"RelyingParty": "http://auth.xboxlive.com",
		"TokenType":    "JWT",
	}
	return doXboxRequest(xblURL, body, "xbl")
}

func xsts(xblToken string) (*xblAuth, error) {
	body := map[string]any{
		"Properties": map[string]any{
			"UserTokens": []string{xblToken},
			"SandboxId":  "RETAIL",
		},
		"RelyingParty": "rp://api.minecraftservices.com/",
		"TokenType":    "JWT",
	}
	return doXboxRequest(xstsURL, body, "xsts")
}

func doXboxRequest(endpoint string, body map[string]any, stage string) (*xblAuth, error) {
	buf, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", endpoint, bytes.NewReader(buf))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := util.Client.Do(req)
	if err != nil {
		return nil, &Error{Stage: stage, Cause: err}
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == 401 && endpoint == xstsURL {
		var j map[string]any
		_ = json.Unmarshal(raw, &j)
		xerr, _ := j["XErr"].(float64)
		return nil, &Error{Stage: stage, Cause: fmt.Errorf("xsts XErr=%d: %s", int64(xerr), describeXErr(int64(xerr)))}
	}
	if resp.StatusCode != 200 {
		return nil, &Error{Stage: stage, Status: resp.StatusCode, Body: string(raw)}
	}
	var j map[string]any
	if err := json.Unmarshal(raw, &j); err != nil {
		return nil, &Error{Stage: stage, Cause: err}
	}
	token, _ := j["Token"].(string)
	dc, _ := j["DisplayClaims"].(map[string]any)
	xui, _ := dc["xui"].([]any)
	if len(xui) == 0 {
		return nil, &Error{Stage: stage, Cause: errors.New("missing DisplayClaims.xui")}
	}
	first, _ := xui[0].(map[string]any)
	uhs, _ := first["uhs"].(string)
	return &xblAuth{Token: token, UserHash: uhs}, nil
}

func describeXErr(x int64) string {
	switch x {
	case 0x8015DC04, 0x8015DC09:
		return "child account; must be added to a Family"
	case 0x8015DC0E:
		return "account not linked to Xbox profile"
	default:
		return "see Xbox XErr documentation"
	}
}
