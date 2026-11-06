package pipeline

import (
	"fmt"
	"image"
	"image/draw"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
)

var srgbLUT [256]float32

func init() {
	for i := range srgbLUT {
		srgbLUT[i] = float32(SRGBToLinear(float64(i) / 255))
	}
}

// Decode reads one image frame (GIF uses its first frame) and converts it to
// linear light. Transparent pixels are composited over black.
func Decode(r io.Reader) (*LinearImage, error) {
	src, _, err := image.Decode(r)
	if err != nil {
		return nil, fmt.Errorf("decode image: %w", err)
	}
	return FromImage(src), nil
}

// FromImage converts any image.Image to linear light.
func FromImage(src image.Image) *LinearImage {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	n := w * h
	out := &LinearImage{
		Width: w, Height: h,
		R: make([]float32, n),
		G: make([]float32, n),
		B: make([]float32, n),
		L: make([]float32, n),
	}
	if w == 0 || h == 0 {
		return out
	}

	nrgba, ok := src.(*image.NRGBA)
	if !ok || b.Min != (image.Point{}) {
		dst := image.NewNRGBA(image.Rect(0, 0, w, h))
		draw.Draw(dst, dst.Bounds(), src, b.Min, draw.Src)
		nrgba = dst
	}

	for y := 0; y < h; y++ {
		srcRow := nrgba.Pix[y*nrgba.Stride : y*nrgba.Stride+w*4]
		dst := y * w
		for x := 0; x < w; x++ {
			i := x * 4
			a := float32(srcRow[i+3]) / 255
			r := srgbLUT[srcRow[i]] * a
			g := srgbLUT[srcRow[i+1]] * a
			bl := srgbLUT[srcRow[i+2]] * a
			out.R[dst+x] = r
			out.G[dst+x] = g
			out.B[dst+x] = bl
			out.L[dst+x] = Luminance(r, g, bl)
		}
	}
	return out
}
