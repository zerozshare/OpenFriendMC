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
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"
	"github.com/google/uuid"
)

type TokenProvider interface {
	Current() string
	RefreshNow() string
}

type FriendJoinListener func(fromPmid uuid.UUID, payload map[string]any)
type WebRtcListener func(fromPmid uuid.UUID, payload map[string]any)

type Client struct {
	tokens    TokenProvider
	onJoin    FriendJoinListener
	onRtc     atomic.Pointer[WebRtcListener]
	sessionID string
	logger    *slog.Logger

	mu         sync.Mutex
	ws         *websocket.Conn
	connecting bool
	closed     bool
	backoff    time.Duration
	pingCancel context.CancelFunc
	ctx        context.Context
	cancel     context.CancelFunc

	rpcID     atomic.Int64
	pendingMu sync.Mutex
	pending   map[int64]chan json.RawMessage

	readyMu sync.Mutex
	readyCh chan struct{}
}

func NewClient(tokens TokenProvider, onJoin FriendJoinListener, logger *slog.Logger) *Client {
	if logger == nil {
		logger = slog.Default()
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &Client{
		tokens:    tokens,
		onJoin:    onJoin,
		sessionID: uuid.New().String(),
		logger:    logger,
		backoff:   initialBackoff,
		ctx:       ctx,
		cancel:    cancel,
		pending:   map[int64]chan json.RawMessage{},
		readyCh:   make(chan struct{}),
	}
}

func (c *Client) markReady() {
	c.readyMu.Lock()
	defer c.readyMu.Unlock()
	select {
	case <-c.readyCh:
	default:
		close(c.readyCh)
	}
}

func (c *Client) resetReady() {
	c.readyMu.Lock()
	defer c.readyMu.Unlock()
	select {
	case <-c.readyCh:
		c.readyCh = make(chan struct{})
	default:
	}
}

func (c *Client) WaitReady(ctx context.Context) error {
	c.readyMu.Lock()
	ch := c.readyCh
	c.readyMu.Unlock()
	select {
	case <-ch:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *Client) SetWebRtcListener(l WebRtcListener) {
	c.onRtc.Store(&l)
}

func (c *Client) Connect() {
	c.mu.Lock()
	if c.closed || c.ws != nil || c.connecting {
		c.mu.Unlock()
		return
	}
	c.connecting = true
	c.mu.Unlock()
	go c.attemptConnect()
}

func (c *Client) Close() {
	c.mu.Lock()
	c.closed = true
	ws := c.ws
	c.ws = nil
	if c.pingCancel != nil {
		c.pingCancel()
		c.pingCancel = nil
	}
	c.mu.Unlock()
	c.cancel()
	if ws != nil {
		_ = ws.Close(websocket.StatusNormalClosure, "bye")
	}
}

func (c *Client) SendClientMessage(toPmid uuid.UUID, encoded string) error {
	params := []any{nil, toPmid.String(), encoded}
	return c.sendRequest("Signaling_SendClientMessage_v1_0", params)
}

func (c *Client) isClosed() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.closed
}

func (c *Client) safeCall(fn func(), label string) {
	defer func() {
		if r := recover(); r != nil {
			c.logger.Warn(label+" panicked", "panic", r)
		}
	}()
	fn()
}
