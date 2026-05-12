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
	"time"

	"github.com/coder/websocket"
)

type rpcMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   json.RawMessage `json:"error,omitempty"`
}

func (c *Client) sendNotification(method string, params any) error {
	body := map[string]any{
		"jsonrpc": "2.0",
		"method":  method,
	}
	if params != nil {
		body["params"] = params
	}
	return c.sendJSON(body)
}

func (c *Client) sendRequest(method string, params any) error {
	id := c.rpcID.Add(1)
	body := map[string]any{
		"jsonrpc": "2.0",
		"method":  method,
		"id":      id,
	}
	if params != nil {
		body["params"] = params
	}
	return c.sendJSON(body)
}

func (c *Client) sendRequestAwait(ctx context.Context, method string, params any) (json.RawMessage, error) {
	id := c.rpcID.Add(1)
	ch := make(chan json.RawMessage, 1)
	c.pendingMu.Lock()
	c.pending[id] = ch
	c.pendingMu.Unlock()

	body := map[string]any{
		"jsonrpc": "2.0",
		"method":  method,
		"id":      id,
	}
	if params != nil {
		body["params"] = params
	}
	if err := c.sendJSON(body); err != nil {
		c.pendingMu.Lock()
		delete(c.pending, id)
		c.pendingMu.Unlock()
		return nil, err
	}
	select {
	case raw, ok := <-ch:
		if !ok {
			return nil, errors.New("rpc returned error")
		}
		return raw, nil
	case <-ctx.Done():
		c.pendingMu.Lock()
		delete(c.pending, id)
		c.pendingMu.Unlock()
		return nil, ctx.Err()
	}
}

func (c *Client) sendResponse(id json.RawMessage, result json.RawMessage) error {
	body := map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"result":  json.RawMessage(result),
	}
	return c.sendJSON(body)
}

func (c *Client) sendJSON(v any) error {
	raw, err := json.Marshal(v)
	if err != nil {
		return err
	}
	c.mu.Lock()
	ws := c.ws
	c.mu.Unlock()
	if ws == nil {
		return errors.New("ws not connected")
	}
	ctx, cancel := context.WithTimeout(c.ctx, 5*time.Second)
	defer cancel()
	return ws.Write(ctx, websocket.MessageText, raw)
}
