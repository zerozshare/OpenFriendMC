/*
 * OpenFriend — Minecraft Java Edition Friends List bridge.
 * Copyright (c) 2026 ZSHARE (https://zpw.jp). Licensed under the MIT License.
 */
package ipc

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"os"
	"sync"
)

type MethodFunc func(ctx context.Context, params json.RawMessage) (any, *RPCError)

type Server struct {
	in      io.Reader
	writer  *Writer
	logger  *slog.Logger
	methods map[string]MethodFunc

	quitOnce sync.Once
	quitCh   chan struct{}
}

func NewServer(in io.Reader, w *Writer, logger *slog.Logger) *Server {
	return &Server{
		in:      in,
		writer:  w,
		logger:  logger,
		methods: map[string]MethodFunc{},
		quitCh:  make(chan struct{}),
	}
}

func (s *Server) Register(name string, fn MethodFunc) {
	s.methods[name] = fn
}

func (s *Server) Writer() *Writer { return s.writer }

func (s *Server) Quit() {
	s.quitOnce.Do(func() { close(s.quitCh) })
}

func (s *Server) Done() <-chan struct{} { return s.quitCh }

func (s *Server) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		select {
		case <-ctx.Done():
		case <-s.quitCh:
			cancel()
		}
	}()

	var wg sync.WaitGroup
	scanner := bufio.NewScanner(s.in)
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		buf := make([]byte, len(line))
		copy(buf, line)
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.dispatch(ctx, buf)
		}()
		select {
		case <-ctx.Done():
			wg.Wait()
			return nil
		default:
		}
	}
	wg.Wait()
	if err := scanner.Err(); err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	return nil
}

func (s *Server) dispatch(ctx context.Context, raw []byte) {
	var req Request
	if err := json.Unmarshal(raw, &req); err != nil {
		_ = s.writer.RespondError(nil, newErrorf(CodeParseError, "parse error", err.Error()))
		return
	}
	if req.JSONRPC != "2.0" {
		_ = s.writer.RespondError(req.ID, newError(CodeInvalidRequest, "jsonrpc must be \"2.0\""))
		return
	}
	fn, ok := s.methods[req.Method]
	if !ok {
		_ = s.writer.RespondError(req.ID, newError(CodeMethodNotFound, "method not found: "+req.Method))
		return
	}
	defer func() {
		if r := recover(); r != nil {
			s.logger.Error("ipc method panic", "method", req.Method, "recover", r)
			_ = s.writer.RespondError(req.ID, newErrorf(CodeInternalError, "internal panic", r))
		}
	}()
	result, rerr := fn(ctx, req.Params)
	if req.ID == nil {
		return
	}
	if rerr != nil {
		_ = s.writer.RespondError(req.ID, rerr)
		return
	}
	if result == nil {
		result = map[string]any{}
	}
	_ = s.writer.Respond(req.ID, result)
}

func ReadStdin() io.Reader { return os.Stdin }
