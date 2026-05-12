/*
 * OpenFriend — Minecraft Java Edition Friends List bridge.
 * Copyright (c) 2026 ZSHARE (https://zpw.jp). Licensed under the MIT License.
 *
 * "Minecraft", "Xbox", "Xbox Live", "Microsoft", and "Mojang" are trademarks
 * of their respective owners. OpenFriend is not affiliated with, endorsed by,
 * sponsored by, or otherwise officially connected to Microsoft Corporation,
 * Mojang AB, or the Xbox brand. See LICENSE for the full notice.
 */
package api

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"

	"jp.zpw.openfriend/internal/util"
)

type SkinVariant string

const (
	SkinClassic SkinVariant = "classic"
	SkinSlim    SkinVariant = "slim"
)

func (c *Client) UploadSkin(pngPath string, variant SkinVariant) error {
	abs, err := filepath.Abs(pngPath)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		return fmt.Errorf("read skin file: %w", err)
	}
	return c.UploadSkinBytes(data, filepath.Base(abs), variant)
}

func (c *Client) UploadSkinBytes(data []byte, filename string, variant SkinVariant) error {
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	if err := mw.WriteField("variant", string(variant)); err != nil {
		return err
	}
	part, err := mw.CreateFormFile("file", filename)
	if err != nil {
		return err
	}
	if _, err := part.Write(data); err != nil {
		return err
	}
	if err := mw.Close(); err != nil {
		return err
	}

	req, _ := http.NewRequest("POST", base+"/minecraft/profile/skins", &body)
	req.Header.Set("Authorization", "Bearer "+c.tokens.Current())
	req.Header.Set("Content-Type", mw.FormDataContentType())

	resp, err := util.Client.Do(req)
	if err != nil {
		return fmt.Errorf("skin upload: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == 401 {
		c.tokens.RefreshNow()
		return c.UploadSkinBytes(data, filename, variant)
	}
	if resp.StatusCode >= 400 {
		return &APIError{Op: "POST /minecraft/profile/skins", Status: resp.StatusCode, Body: string(raw)}
	}
	return nil
}
