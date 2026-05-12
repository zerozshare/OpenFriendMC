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
	"strings"

	"github.com/google/uuid"
)

func (c *Client) handleMessage(raw []byte) {
	var msg rpcMessage
	if err := json.Unmarshal(raw, &msg); err != nil {
		c.logger.Warn("WS parse failed", "err", err, "raw", string(raw))
		return
	}
	if msg.Method != "" {
		c.handleIncomingMethod(msg.ID, msg.Method, msg.Params)
		return
	}
	if len(msg.ID) > 0 && (len(msg.Result) > 0 || len(msg.Error) > 0) {
		var idNum int64
		if json.Unmarshal(msg.ID, &idNum) != nil {
			return
		}
		c.pendingMu.Lock()
		ch, ok := c.pending[idNum]
		if ok {
			delete(c.pending, idNum)
		}
		c.pendingMu.Unlock()
		if ok {
			if len(msg.Error) > 0 {
				close(ch)
			} else {
				ch <- msg.Result
				close(ch)
			}
		}
	}
}

func (c *Client) handleIncomingMethod(id json.RawMessage, method string, params json.RawMessage) {
	switch method {
	case "System_Pong_v1_0":
		return
	case "Signaling_ReceiveMessage_v1_0":
		c.handleReceiveMessage(id, params)
	default:
		c.logger.Debug("Unhandled WS method", "method", method)
	}
}

func (c *Client) handleReceiveMessage(id json.RawMessage, params json.RawMessage) {
	if len(id) > 0 && string(id) != "null" {
		_ = c.sendResponse(id, json.RawMessage(`{}`))
	}
	var arr []map[string]any
	if err := json.Unmarshal(params, &arr); err != nil || len(arr) == 0 {
		return
	}
	envelope := arr[0]
	from, _ := envelope["From"].(string)
	inner, _ := envelope["Message"].(string)
	fromPmid, err := uuid.Parse(from)
	if err != nil {
		return
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(inner), &payload); err != nil {
		return
	}
	typ, _ := payload["type"].(string)
	if strings.HasPrefix(typ, "JOIN_") || typ == "INVITE_DECLINED" {
		if c.onJoin != nil {
			c.safeCall(func() { c.onJoin(fromPmid, payload) }, "FriendJoin listener")
		}
	} else if typ == "OFFER" || typ == "ANSWER" || typ == "ICE_CANDIDATE" {
		if lp := c.onRtc.Load(); lp != nil {
			l := *lp
			c.safeCall(func() { l(fromPmid, payload) }, "WebRtc listener")
		}
	}
}
