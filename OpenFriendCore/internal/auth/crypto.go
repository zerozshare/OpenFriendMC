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
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"

	"github.com/denisbrodbeck/machineid"
)

const (
	cipherMachineBound = "AES-256-GCM-MachineBound"
	cipherAppID        = "openfriend"
)

var (
	ErrMachineKeyUnavailable = errors.New("machine ID unavailable on this system")
	ErrWrongMachine          = errors.New("credentials file is bound to a different machine")
)

func machineKey() ([]byte, error) {
	id, err := machineid.ProtectedID(cipherAppID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrMachineKeyUnavailable, err)
	}
	raw, err := hex.DecodeString(id)
	if err != nil || len(raw) < 32 {
		return nil, ErrMachineKeyUnavailable
	}
	return raw[:32], nil
}

func encryptMachineBound(plaintext []byte) (*pem.Block, error) {
	key, err := machineKey()
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	ct := gcm.Seal(nil, nonce, plaintext, nil)
	return &pem.Block{
		Type: pemType,
		Headers: map[string]string{
			"Cipher": cipherMachineBound,
			"Nonce":  base64.StdEncoding.EncodeToString(nonce),
		},
		Bytes: ct,
	}, nil
}

func decryptMachineBound(block *pem.Block) ([]byte, error) {
	nonceB64, ok := block.Headers["Nonce"]
	if !ok {
		return nil, errors.New("missing Nonce header")
	}
	nonce, err := base64.StdEncoding.DecodeString(nonceB64)
	if err != nil {
		return nil, fmt.Errorf("invalid Nonce: %w", err)
	}
	key, err := machineKey()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrWrongMachine, err)
	}
	blk, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(blk)
	if err != nil {
		return nil, err
	}
	plain, err := gcm.Open(nil, nonce, block.Bytes, nil)
	if err != nil {
		return nil, fmt.Errorf("%w (file may have been copied from another machine or tampered with)", ErrWrongMachine)
	}
	return plain, nil
}
