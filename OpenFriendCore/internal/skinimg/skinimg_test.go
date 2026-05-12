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
	"image/png"
	"testing"
)

func TestPrepareFace(t *testing.T) {
	res, err := Prepare("/tmp/face32.png")
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	if !res.Processed {
		t.Fatalf("expected processed=true for 32x32 input")
	}
	img, err := png.Decode(bytes.NewReader(res.PNG))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	b := img.Bounds()
	if b.Dx() != 64 || b.Dy() != 64 {
		t.Fatalf("expected 64x64, got %dx%d", b.Dx(), b.Dy())
	}
	_, _, _, a := img.At(8, 8).RGBA()
	if a == 0 {
		t.Fatalf("face area (8,8) should be opaque, got alpha=0")
	}
	_, _, _, a = img.At(0, 0).RGBA()
	if a != 0 {
		t.Fatalf("non-face area (0,0) should be transparent, got alpha=%d", a)
	}
}
