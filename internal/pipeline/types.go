// Package pipeline converts a decoded image into normalized luminance grids.
//
// Every stage is a pure function over the types in this file. That keeps the
// expensive parts testable without a terminal, a font, or a renderer.
package pipeline

import "math"

// LinearImage is an sRGB image converted to linear light, with alpha composited
// over black. R, G, B and L all have len == Width*Height.
type LinearImage struct {
	Width, Height int
	R, G, B, L    []float32
}

// Grid is a rectangular sample grid. L is normalized luminance in [0,1].
// R, G and B are the matching average linear-light colours.
type Grid struct {
	Cols, Rows int
	L          []float32
	R, G, B    []float32
}

// At returns the luminance at x, y.
func (g *Grid) At(x, y int) float32 {
	return g.L[y*g.Cols+x]
}

// SRGBToLinear converts one sRGB channel in [0,1] to linear light.
func SRGBToLinear(v float64) float64 {
	if v <= 0.04045 {
		return v / 12.92
	}
	return math.Pow((v+0.055)/1.055, 2.4)
}

// LinearToSRGB converts one linear-light channel in [0,1] back to sRGB.
func LinearToSRGB(v float64) float64 {
	if v <= 0.0031308 {
		return 12.92 * v
	}
	return 1.055*math.Pow(v, 1/2.4) - 0.055
}

// Luminance computes linear-light relative luminance.
func Luminance(r, g, b float32) float32 {
	return 0.2126*r + 0.7152*g + 0.0722*b
}

func clamp01(v float32) float32 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
