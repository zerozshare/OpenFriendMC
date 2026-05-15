/*
 * OpenFriend — Minecraft Java Edition Friends List bridge.
 * Copyright (c) 2026 ZSHARE (https://zpw.jp). Licensed under the MIT License.
 *
 * "Minecraft", "Xbox", "Xbox Live", "Microsoft", and "Mojang" are trademarks
 * of their respective owners. OpenFriend is not affiliated with, endorsed by,
 * sponsored by, or otherwise officially connected to Microsoft Corporation,
 * Mojang AB, or the Xbox brand. See LICENSE for the full notice.
 */
package skinimg

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"

	xdraw "golang.org/x/image/draw"
)

const (
	skinW, skinH = 64, 64
	faceX, faceY = 8, 8
	faceSize     = 8
)

type Result struct {
	PNG       []byte
	Processed bool
	OriginalW int
	OriginalH int
}

func Prepare(path string) (*Result, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read skin file: %w", err)
	}
	img, err := png.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("decode PNG: %w", err)
	}
	w := img.Bounds().Dx()
	h := img.Bounds().Dy()

	if (w == 64 && h == 64) || (w == 64 && h == 32) {
		return &Result{PNG: raw, OriginalW: w, OriginalH: h}, nil
	}
	if w != h {
		return nil, fmt.Errorf("skin must be 64x64, 64x32, or square face-only (got %dx%d)", w, h)
	}

	composed := composeFromFace(img)
	var out bytes.Buffer
	if err := png.Encode(&out, composed); err != nil {
		return nil, err
	}
	return &Result{PNG: out.Bytes(), Processed: true, OriginalW: w, OriginalH: h}, nil
}

func composeFromFace(face image.Image) *image.NRGBA {
	canvas := image.NewNRGBA(image.Rect(0, 0, skinW, skinH))
	draw.Draw(canvas, canvas.Bounds(), &image.Uniform{C: color.NRGBA{0, 0, 0, 0}}, image.Point{}, draw.Src)

	faceArea := image.Rect(faceX, faceY, faceX+faceSize, faceY+faceSize)
	xdraw.CatmullRom.Scale(canvas, faceArea, face, face.Bounds(), xdraw.Over, nil)
	hardenAlpha(canvas, faceArea)
	return canvas
}

func hardenAlpha(img *image.NRGBA, area image.Rectangle) {
	for y := area.Min.Y; y < area.Max.Y; y++ {
		for x := area.Min.X; x < area.Max.X; x++ {
			c := img.NRGBAAt(x, y)
			if c.A == 0 {
				continue
			}
			if c.A < 255 {
				c.A = 255
				img.SetNRGBA(x, y, c)
			}
		}
	}
}

var _ = errors.New
