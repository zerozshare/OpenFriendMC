/*
 * OpenFriend — Minecraft Java Edition Friends List bridge.
 * Copyright (c) 2026 ZSHARE (https://zpw.jp). Licensed under the MIT License.
 *
 * "Minecraft", "Xbox", "Xbox Live", "Microsoft", and "Mojang" are trademarks
 * of their respective owners. OpenFriend is not affiliated with, endorsed by,
 * sponsored by, or otherwise officially connected to Microsoft Corporation,
 * Mojang AB, or the Xbox brand. See LICENSE for the full notice.
 */
package bridge

import (
	"errors"
	"io"
	"log/slog"
	"net"
	"sync"
	"time"
)

type TCPBridge struct {
	conn   net.Conn
	logger *slog.Logger

	closeOnce sync.Once
	onClose   func()
}

func DialTCP(addr string, onDownstream func([]byte), onClose func(), logger *slog.Logger) (*TCPBridge, error) {
	if logger == nil {
		logger = slog.Default()
	}
	d := net.Dialer{Timeout: 5 * time.Second}
	conn, err := d.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	if tcp, ok := conn.(*net.TCPConn); ok {
		_ = tcp.SetNoDelay(true)
	}
	b := &TCPBridge{conn: conn, logger: logger, onClose: onClose}
	go b.readLoop(onDownstream)
	return b, nil
}

func (b *TCPBridge) readLoop(onDownstream func([]byte)) {
	defer b.Close()
	buf := make([]byte, 16*1024)
	for {
		n, err := b.conn.Read(buf)
		if n > 0 {
			chunk := make([]byte, n)
			copy(chunk, buf[:n])
			onDownstream(chunk)
		}
		if err != nil {
			if !errors.Is(err, io.EOF) {
				b.logger.Debug("TCP read ended", "err", err)
			}
			return
		}
	}
}

func (b *TCPBridge) Feed(data []byte) error {
	_, err := b.conn.Write(data)
	return err
}

func (b *TCPBridge) Close() {
	b.closeOnce.Do(func() {
		_ = b.conn.Close()
		if b.onClose != nil {
			b.onClose()
		}
	})
}

func ProbeTCP(addr string, timeout time.Duration) bool {
	d := net.Dialer{Timeout: timeout}
	conn, err := d.Dial("tcp", addr)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}
