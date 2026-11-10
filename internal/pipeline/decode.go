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
	return DecodeWithBackground(r, "black")
}

// DecodeWithBackground decodes an image and composites transparent pixels over
// the requested background. background is "black", "white", or "auto"; auto
// uses white when the image contains transparency and black otherwise.
func DecodeWithBackground(r io.Reader, background string) (*LinearImage, error) {
	src, _, err := image.Decode(r)
	if err != nil {
		return nil, fmt.Errorf("decode image: %w", err)
	}
	return FromImageWithBackground(src, background), nil
}

// FromImage converts any image.Image to linear light over a black background.
func FromImage(src image.Image) *LinearImage {
	return FromImageWithBackground(src, "black")
}

// FromImageWithBackground converts any image.Image to linear light and
// composites transparent pixels over black or white.
func FromImageWithBackground(src image.Image, background string) *LinearImage {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	n := w * h
	out := &LinearImage{
		Width: w, Height: h,
		R: make([]float32, n),
		G: make([]float32, n),
		B: make([]float32, n),
		L: make([]float32, n),
		A: make([]float32, n),
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

	white := background == "white"
	if background == "auto" {
		white = hasTransparency(nrgba) && autoPrefersWhite(nrgba)
	}
	var bg float32
	if white {
		bg = 1
		out.Background = "white"
	} else {
		out.Background = "black"
	}

	for y := 0; y < h; y++ {
		srcRow := nrgba.Pix[y*nrgba.Stride : y*nrgba.Stride+w*4]
		dst := y * w
		for x := 0; x < w; x++ {
			i := x * 4
			a := float32(srcRow[i+3]) / 255
			inv := 1 - a
			r := srgbLUT[srcRow[i]]*a + bg*inv
			g := srgbLUT[srcRow[i+1]]*a + bg*inv
			bl := srgbLUT[srcRow[i+2]]*a + bg*inv
			out.R[dst+x] = r
			out.G[dst+x] = g
			out.B[dst+x] = bl
			out.L[dst+x] = Luminance(r, g, bl)
			out.A[dst+x] = a
		}
	}
	return out
}

func hasTransparency(img *image.NRGBA) bool {
	w, h := img.Rect.Dx(), img.Rect.Dy()
	for y := 0; y < h; y++ {
		row := img.Pix[y*img.Stride : y*img.Stride+w*4]
		for i := 3; i < len(row); i += 4 {
			if row[i] < 250 {
				return true
			}
		}
	}
	return false
}

// autoPrefersWhite averages the visible subject. A dark subject gets a white
// backdrop and a light subject gets black, so transparent logos keep their edge
// instead of disappearing into a same-coloured page.
func autoPrefersWhite(img *image.NRGBA) bool {
	w, h := img.Rect.Dx(), img.Rect.Dy()
	var sum, weight float64
	for y := 0; y < h; y++ {
		row := img.Pix[y*img.Stride : y*img.Stride+w*4]
		for i := 0; i+3 < len(row); i += 4 {
			a := float64(row[i+3]) / 255
			if a < 0.5 {
				continue
			}
			l := (0.2126*float64(row[i]) + 0.7152*float64(row[i+1]) + 0.0722*float64(row[i+2])) / 255
			sum += l * a
			weight += a
		}
	}
	if weight == 0 {
		return false
	}
	return sum/weight < 0.5
}
