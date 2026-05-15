/*
 * OpenFriend — Minecraft Java Edition Friends List bridge.
 * Copyright (c) 2026 ZSHARE (https://zpw.jp). Licensed under the MIT License.
 */
package ipc

import (
	"bufio"
	"encoding/json"
	"io"
	"sync"
)

type Writer struct {
	mu  sync.Mutex
	out *bufio.Writer
}

func NewWriter(w io.Writer) *Writer {
	return &Writer{out: bufio.NewWriter(w)}
}

func (w *Writer) write(v any) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	buf, err := json.Marshal(v)
	if err != nil {
		return err
	}
	if _, err := w.out.Write(buf); err != nil {
		return err
	}
	if err := w.out.WriteByte('\n'); err != nil {
		return err
	}
	return w.out.Flush()
}

func (w *Writer) Respond(id json.RawMessage, result any) error {
	return w.write(Response{JSONRPC: "2.0", ID: id, Result: result})
}

func (w *Writer) RespondError(id json.RawMessage, err *RPCError) error {
	return w.write(Response{JSONRPC: "2.0", ID: id, Error: err})
}

func (w *Writer) Notify(method string, params any) error {
	return w.write(Notification{JSONRPC: "2.0", Method: method, Params: params})
}
