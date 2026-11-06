package render

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// WritePNG renders the art with the same monospace font used for matching.
func WritePNG(w io.Writer, art *Art, fontData []byte, cellW, cellH int) error {
	parsed, err := opentype.Parse(fontData)
	if err != nil {
		return fmt.Errorf("parse font: %w", err)
	}
	face, err := opentype.NewFace(parsed, &opentype.FaceOptions{
		Size:    float64(cellH) * 0.86,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return fmt.Errorf("create face: %w", err)
	}
	defer face.Close()

	dst := image.NewRGBA(image.Rect(0, 0, art.Cols*cellW, art.Rows*cellH))
	draw.Draw(dst, dst.Bounds(), image.NewUniform(color.RGBA{0, 0, 0, 255}), image.Point{}, draw.Src)
	metrics := face.Metrics()
	baseline := (cellH - (metrics.Ascent.Ceil() + metrics.Descent.Ceil())) / 2
	baseline += metrics.Ascent.Ceil()

	for y := 0; y < art.Rows; y++ {
		for x := 0; x < art.Cols; x++ {
			i := y*art.Cols + x
			fg := color.RGBA{235, 235, 235, 255}
			if !art.Monochrome && art.FG[i] != (Color{}) {
				fg = color.RGBA{art.FG[i].R, art.FG[i].G, art.FG[i].B, 255}
			}
			if art.Halfblock {
				bg := art.BG[i]
				rect := image.Rect(x*cellW, y*cellH+cellH/2, (x+1)*cellW, (y+1)*cellH)
				draw.Draw(dst, rect, image.NewUniform(color.RGBA{bg.R, bg.G, bg.B, 255}), image.Point{}, draw.Src)
			}
			d := font.Drawer{
				Dst:  dst,
				Src:  image.NewUniform(fg),
				Face: face,
				Dot:  fixed.P(x*cellW, y*cellH+baseline),
			}
			d.DrawString(string(art.Chars[i]))
		}
	}
	return png.Encode(w, dst)
}
