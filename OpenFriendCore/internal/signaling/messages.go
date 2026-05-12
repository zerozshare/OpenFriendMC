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
	"encoding/json"
)

func JoinRequest(sessionID string) string {
	b, _ := json.Marshal(map[string]any{
		"type":      "JOIN_REQUEST",
		"sessionId": sessionID,
	})
	return string(b)
}

func Offer(sessionID, sdp string) string {
	b, _ := json.Marshal(map[string]any{
		"type":      "OFFER",
		"sessionId": sessionID,
		"sdp":       sdp,
	})
	return string(b)
}

func JoinAccepted(sessionID string) string {
	b, _ := json.Marshal(map[string]any{
		"type":      "JOIN_ACCEPTED",
		"sessionId": sessionID,
	})
	return string(b)
}

func JoinRejected(sessionID string) string {
	b, _ := json.Marshal(map[string]any{
		"type":      "JOIN_REJECTED",
		"sessionId": sessionID,
	})
	return string(b)
}

func Answer(sessionID, sdp string) string {
	b, _ := json.Marshal(map[string]any{
		"type":      "ANSWER",
		"sessionId": sessionID,
		"sdp":       sdp,
	})
	return string(b)
}

type IceCandidatePayload struct {
	Candidate     string `json:"candidate"`
	SdpMid        string `json:"sdpMid"`
	SdpMLineIndex int    `json:"sdpMLineIndex"`
}

func IceCandidate(sessionID, candidate, sdpMid string, sdpMLineIndex int) string {
	b, _ := json.Marshal(map[string]any{
		"type":      "ICE_CANDIDATE",
		"sessionId": sessionID,
		"iceCandidate": IceCandidatePayload{
			Candidate:     candidate,
			SdpMid:        sdpMid,
			SdpMLineIndex: sdpMLineIndex,
		},
	})
	return string(b)
}

func ParseIceCandidate(payload map[string]any) (IceCandidatePayload, bool) {
	raw, ok := payload["iceCandidate"]
	if !ok {
		return IceCandidatePayload{}, false
	}
	m, ok := raw.(map[string]any)
	if !ok {
		return IceCandidatePayload{}, false
	}
	out := IceCandidatePayload{}
	out.Candidate, _ = m["candidate"].(string)
	if v, ok := m["sdpMid"].(string); ok {
		out.SdpMid = v
	} else {
		out.SdpMid = "0"
	}
	if v, ok := m["sdpMLineIndex"].(float64); ok {
		out.SdpMLineIndex = int(v)
	}
	return out, true
}

func GetSessionID(payload map[string]any) string {
	if v, ok := payload["sessionId"].(string); ok {
		return v
	}
	return ""
}

func GetSDP(payload map[string]any) string {
	if v, ok := payload["sdp"].(string); ok {
		return v
	}
	return ""
}
