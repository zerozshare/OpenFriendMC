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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"jp.zpw.openfriend/internal/util"
)

func StartDeviceCode() (*DeviceCodePrompt, error) {
	form := url.Values{}
	form.Set("client_id", clientID)
	form.Set("scope", scope)
	form.Set("response_type", "device_code")

	res, err := postForm(deviceCodeURL, form)
	if err != nil {
		return nil, &Error{Stage: "device-code", Cause: err}
	}
	if res.statusCode != 200 {
		return nil, &Error{Stage: "device-code", Status: res.statusCode, Body: res.body}
	}
	var j map[string]any
	if err := json.Unmarshal([]byte(res.body), &j); err != nil {
		return nil, &Error{Stage: "device-code", Cause: err}
	}
	userCode, _ := j["user_code"].(string)
	deviceCode, _ := j["device_code"].(string)
	expiresIn, _ := j["expires_in"].(float64)
	interval, _ := j["interval"].(float64)

	verifyURI, _ := j["verification_uri"].(string)
	if verifyURI == "" {
		verifyURI, _ = j["verification_url"].(string)
	}
	msg, _ := j["message"].(string)
	if msg == "" {
		msg = fmt.Sprintf("Visit %s and enter %s", verifyURI, userCode)
	}

	return &DeviceCodePrompt{
		UserCode:        userCode,
		VerificationURI: verifyURI,
		Message:         msg,
		DeviceCode:      deviceCode,
		ExpiresIn:       time.Duration(expiresIn) * time.Second,
		Interval:        time.Duration(interval) * time.Second,
	}, nil
}

func PollForMsToken(prompt *DeviceCodePrompt, onWait func(msg string)) (*MsToken, error) {
	deadline := time.Now().Add(prompt.ExpiresIn)
	interval := prompt.Interval
	for time.Now().Before(deadline) {
		time.Sleep(interval)

		form := url.Values{}
		form.Set("grant_type", "urn:ietf:params:oauth:grant-type:device_code")
		form.Set("client_id", clientID)
		form.Set("device_code", prompt.DeviceCode)

		res, err := postForm(tokenURL, form)
		if err != nil {
			return nil, &Error{Stage: "device-poll", Cause: err}
		}
		var j map[string]any
		_ = json.Unmarshal([]byte(res.body), &j)
		if res.statusCode == 200 {
			accessToken, _ := j["access_token"].(string)
			refreshToken, _ := j["refresh_token"].(string)
			expiresIn, _ := j["expires_in"].(float64)
			return &MsToken{
				AccessToken:  accessToken,
				RefreshToken: refreshToken,
				ExpiresAt:    time.Now().Add(time.Duration(expiresIn) * time.Second),
			}, nil
		}
		errCode, _ := j["error"].(string)
		switch errCode {
		case "authorization_pending":
			if onWait != nil {
				onWait("waiting for approval...")
			}
		case "slow_down":
			interval += 5 * time.Second
			if onWait != nil {
				onWait("slowing down poll")
			}
		case "expired_token", "expired_device_code":
			return nil, &Error{Stage: "device-poll", Cause: errors.New("device code expired")}
		case "authorization_declined":
			return nil, &Error{Stage: "device-poll", Cause: errors.New("user declined")}
		default:
			return nil, &Error{Stage: "device-poll", Status: res.statusCode, Body: res.body}
		}
	}
	return nil, &Error{Stage: "device-poll", Cause: errors.New("timed out")}
}

func RefreshMsToken(refreshToken string) (*MsToken, error) {
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("client_id", clientID)
	form.Set("refresh_token", refreshToken)
	form.Set("scope", scope)

	res, err := postForm(tokenURL, form)
	if err != nil {
		return nil, &Error{Stage: "ms-refresh", Cause: err}
	}
	if res.statusCode != 200 {
		return nil, &Error{Stage: "ms-refresh", Status: res.statusCode, Body: res.body}
	}
	var j map[string]any
	if err := json.Unmarshal([]byte(res.body), &j); err != nil {
		return nil, &Error{Stage: "ms-refresh", Cause: err}
	}
	accessToken, _ := j["access_token"].(string)
	newRefresh, _ := j["refresh_token"].(string)
	expiresIn, _ := j["expires_in"].(float64)
	return &MsToken{
		AccessToken:  accessToken,
		RefreshToken: newRefresh,
		ExpiresAt:    time.Now().Add(time.Duration(expiresIn) * time.Second),
	}, nil
}

type httpResp struct {
	statusCode int
	body       string
}

func postForm(endpoint string, form url.Values) (*httpResp, error) {
	req, _ := http.NewRequest("POST", endpoint, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := util.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	return &httpResp{statusCode: resp.StatusCode, body: string(raw)}, nil
}
