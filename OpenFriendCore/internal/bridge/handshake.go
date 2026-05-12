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
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

const (
	maxVarInt           = 5
	maxAddressLen       = 255
	bypassMarkerSegment = "openfriend"
	bypassNonceBytes    = 12
)

func writeVarInt(w *bytes.Buffer, v int32) {
	u := uint32(v)
	for {
		b := byte(u & 0x7F)
		u >>= 7
		if u != 0 {
			b |= 0x80
			w.WriteByte(b)
		} else {
			w.WriteByte(b)
			return
		}
	}
}

func readVarInt(r io.ByteReader) (int32, int, error) {
	var value uint32
	for i := 0; i < maxVarInt; i++ {
		b, err := r.ReadByte()
		if err != nil {
			return 0, i, err
		}
		value |= uint32(b&0x7F) << (7 * i)
		if b&0x80 == 0 {
			return int32(value), i + 1, nil
		}
	}
	return 0, maxVarInt, errors.New("varint overflow")
}

func writeString(w *bytes.Buffer, s string) {
	writeVarInt(w, int32(len(s)))
	w.WriteString(s)
}

func readString(r *bytes.Reader, maxLen int) (string, error) {
	n, _, err := readVarInt(r)
	if err != nil {
		return "", err
	}
	if n < 0 || int(n) > maxLen {
		return "", fmt.Errorf("string length out of range: %d", n)
	}
	buf := make([]byte, n)
	if _, err := io.ReadFull(r, buf); err != nil {
		return "", err
	}
	return string(buf), nil
}

type Handshake struct {
	ProtocolVersion int32
	ServerAddress   string
	ServerPort      uint16
	NextState       int32
}

func decodeHandshake(payload []byte) (*Handshake, error) {
	r := bytes.NewReader(payload)
	packetID, _, err := readVarInt(r)
	if err != nil {
		return nil, fmt.Errorf("packet id: %w", err)
	}
	if packetID != 0 {
		return nil, fmt.Errorf("first packet must be Handshake (id=0), got %d", packetID)
	}
	proto, _, err := readVarInt(r)
	if err != nil {
		return nil, fmt.Errorf("protocol: %w", err)
	}
	addr, err := readString(r, maxAddressLen*2)
	if err != nil {
		return nil, fmt.Errorf("address: %w", err)
	}
	var port uint16
	if err := binary.Read(r, binary.BigEndian, &port); err != nil {
		return nil, fmt.Errorf("port: %w", err)
	}
	next, _, err := readVarInt(r)
	if err != nil {
		return nil, fmt.Errorf("next state: %w", err)
	}
	return &Handshake{ProtocolVersion: proto, ServerAddress: addr, ServerPort: port, NextState: next}, nil
}

func encodeHandshake(h *Handshake) []byte {
	body := bytes.Buffer{}
	writeVarInt(&body, 0)
	writeVarInt(&body, h.ProtocolVersion)
	writeString(&body, h.ServerAddress)
	binary.Write(&body, binary.BigEndian, h.ServerPort)
	writeVarInt(&body, h.NextState)

	framed := bytes.Buffer{}
	writeVarInt(&framed, int32(body.Len()))
	framed.Write(body.Bytes())
	return framed.Bytes()
}

func readFramedPacket(buf []byte) (payload []byte, consumed int, ok bool, err error) {
	r := bytes.NewReader(buf)
	length, lenSize, err := readVarInt(r)
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return nil, 0, false, nil
	}
	if err != nil {
		return nil, 0, false, fmt.Errorf("frame length: %w", err)
	}
	if length < 0 || length > 2*1024*1024 {
		return nil, 0, false, fmt.Errorf("frame length out of range: %d", length)
	}
	total := lenSize + int(length)
	if len(buf) < total {
		return nil, 0, false, nil
	}
	return buf[lenSize:total], total, true, nil
}

func InjectBypassMarker(address string, key []byte) (string, error) {
	nonce := make([]byte, bypassNonceBytes)
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	nonceB64 := base64.RawStdEncoding.EncodeToString(nonce)
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(nonceB64))
	sig := base64.RawStdEncoding.EncodeToString(mac.Sum(nil))
	out := address + "\x00" + bypassMarkerSegment + "\x00" + nonceB64 + "\x00" + sig
	if len(out) > maxAddressLen {
		return "", fmt.Errorf("address with marker exceeds %d bytes (got %d)", maxAddressLen, len(out))
	}
	return out, nil
}
