package pipeline

import (
	"image"
	"image/color"
)

// NewTestImage returns a deterministic LRGBA image with a usable non-zero
// sub-image so tests exercise bounds handling.
func NewTestImage(w, h int) image.Image {
	full := image.NewNRGBA(image.Rect(0, 0, w+2, h+2))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			full.SetNRGBA(x+1, y+1, color.NRGBA{R: uint8(x * 50), G: uint8(y * 50), B: 128, A: 255})
		}
	}
	return full.SubImage(image.Rect(1, 1, 1+w, 1+h))
}
