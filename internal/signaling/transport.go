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
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/coder/websocket"
	"github.com/google/uuid"
	"jp.zpw.openfriend/internal/util"
)

func (c *Client) attemptConnect() {
	if c.isClosed() {
		return
	}
	err := c.doConnect()
	c.mu.Lock()
	if err == nil {
		c.backoff = initialBackoff
		c.connecting = false
		c.mu.Unlock()
		return
	}
	c.mu.Unlock()
	c.logger.Warn("Signaling connect failed", "err", err)
	c.scheduleReconnect(err)
}

func (c *Client) scheduleReconnect(cause error) {
	if c.isClosed() {
		return
	}
	c.mu.Lock()
	c.ws = nil
	c.connecting = true
	delay := c.backoff
	c.backoff *= 2
	if c.backoff > maxBackoff {
		c.backoff = maxBackoff
	}
	c.mu.Unlock()

	if cause != nil && strings.Contains(cause.Error(), "status=401") {
		c.tokens.RefreshNow()
	}
	c.logger.Info("Reconnecting signaling", "delay", delay)
	go func() {
		select {
		case <-time.After(delay):
			c.attemptConnect()
		case <-c.ctx.Done():
		}
	}()
}

func (c *Client) doConnect() error {
	wsBase, err := c.fetchSignalingURI()
	if err != nil {
		return err
	}
	requestID := uuid.New().String()
	wsURL := wsBase + wsPath
	c.logger.Info("Opening signaling WebSocket", "url", wsURL)

	hdr := http.Header{}
	hdr.Set("x-mojangauth", c.tokens.Current())
	hdr.Set("Session-Id", c.sessionID)
	hdr.Set("Request-Id", requestID)

	ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, wsURL, &websocket.DialOptions{HTTPHeader: hdr})
	if err != nil {
		return err
	}
	conn.SetReadLimit(8 * 1024 * 1024)

	c.mu.Lock()
	c.ws = conn
	pingCtx, pingCancel := context.WithCancel(c.ctx)
	c.pingCancel = pingCancel
	c.mu.Unlock()

	c.logger.Info("Signaling WebSocket open")
	c.markReady()
	go c.readLoop(conn)
	go c.pingLoop(pingCtx)
	return nil
}

func (c *Client) fetchSignalingURI() (string, error) {
	doGet := func() (*http.Response, error) {
		req, _ := http.NewRequest("GET", base+configPath, nil)
		req.Header.Set("x-mojangauth", c.tokens.Current())
		req.Header.Set("Session-Id", c.sessionID)
		req.Header.Set("Request-Id", uuid.New().String())
		return util.Client.Do(req)
	}
	resp, err := doGet()
	if err != nil {
		return "", err
	}
	if resp.StatusCode == 401 {
		resp.Body.Close()
		c.tokens.RefreshNow()
		resp, err = doGet()
		if err != nil {
			return "", err
		}
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("signaling config failed: status=%d body=%s", resp.StatusCode, string(raw))
	}
	var j map[string]any
	if err := json.Unmarshal(raw, &j); err != nil {
		return "", err
	}
	result, _ := j["result"].(map[string]any)
	uri, _ := result["signalingUri"].(string)
	if uri == "" {
		return "", errors.New("missing signalingUri in config response")
	}
	return uri, nil
}

func (c *Client) pingLoop(ctx context.Context) {
	t := time.NewTicker(pingInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if err := c.sendNotification("System_Ping_v1_0", nil); err != nil {
				c.logger.Warn("Ping failed", "err", err)
			}
		}
	}
}

func (c *Client) readLoop(ws *websocket.Conn) {
	for {
		_, raw, err := ws.Read(c.ctx)
		if err != nil {
			c.logger.Warn("Signaling WS read ended", "err", err)
			c.handleClose(err)
			return
		}
		c.handleMessage(raw)
	}
}

func (c *Client) handleClose(cause error) {
	c.mu.Lock()
	if c.pingCancel != nil {
		c.pingCancel()
		c.pingCancel = nil
	}
	c.ws = nil
	c.mu.Unlock()
	c.resetReady()
	c.scheduleReconnect(cause)
}
