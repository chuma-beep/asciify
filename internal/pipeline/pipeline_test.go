package pipeline

import (
	"image"
	"image/color"
	"math"
	"testing"
)

func TestSRGBToLinearKnownValues(t *testing.T) {
	cases := []struct {
		in   float64
		want float64
	}{
		{0, 0},
		{1, 1},
		{128.0 / 255.0, 0.2158605},
	}
	for _, tc := range cases {
		got := SRGBToLinear(tc.in)
		if math.Abs(got-tc.want) > 1e-5 {
			t.Fatalf("SRGBToLinear(%v) = %v, want %v", tc.in, got, tc.want)
		}
		if back := LinearToSRGB(got); math.Abs(back-tc.in) > 1e-5 {
			t.Fatalf("LinearToSRGB(SRGBToLinear(%v)) = %v", tc.in, back)
		}
	}
}

func TestClipPercentilesAndNormalize(t *testing.T) {
	grid := &Grid{Cols: 100, Rows: 1, L: make([]float32, 100)}
	for i := range grid.L {
		grid.L[i] = float32(i) / 99
	}
	low, high := ClipPercentiles(grid, 0.02, 0.98)
	if low < 0.01 || low > 0.04 {
		t.Fatalf("low = %v, want around 0.02", low)
	}
	if high < 0.96 || high > 0.99 {
		t.Fatalf("high = %v, want around 0.98", high)
	}
	Normalize(grid, low, high)
	if grid.L[0] > 0.001 || grid.L[len(grid.L)-1] < 0.999 {
		t.Fatalf("normalize did not span [0,1]: %v .. %v", grid.L[0], grid.L[len(grid.L)-1])
	}
}

func TestCellRowsPreservesSquareAspect(t *testing.T) {
	if got := CellRows(420, 420, 80, 2); got != 40 {
		t.Fatalf("CellRows = %d, want 40", got)
	}
}

func TestResampleAreaAverage(t *testing.T) {
	img := &LinearImage{
		Width: 2, Height: 2,
		R: []float32{0, 1, 0, 1},
		G: []float32{0, 1, 0, 1},
		B: []float32{0, 1, 0, 1},
		L: []float32{0, 1, 0, 1},
	}
	grid := Resample(img, 1, 1)
	if math.Abs(float64(grid.R[0])-0.5) > 1e-6 {
		t.Fatalf("area average = %v, want 0.5", grid.R[0])
	}
}

func TestDecodePreservesNonZeroBounds(t *testing.T) {
	src := NewTestImage(4, 4)
	img := FromImage(src)
	if img.Width != 4 || img.Height != 4 {
		t.Fatalf("size = %dx%d", img.Width, img.Height)
	}
}

func TestApplyGamma(t *testing.T) {
	grid := &Grid{Cols: 2, Rows: 1, L: []float32{0.25, 1}}
	ApplyGamma(grid, 2)
	if math.Abs(float64(grid.L[0])-0.5) > 1e-6 {
		t.Fatalf("gamma 2 on 0.25 = %v, want 0.5", grid.L[0])
	}
	if grid.L[1] != 1 {
		t.Fatalf("gamma must preserve white, got %v", grid.L[1])
	}

	values := []float32{0.25}
	ApplyGammaValues(values, 0.5)
	if math.Abs(float64(values[0])-0.0625) > 1e-6 {
		t.Fatalf("gamma 0.5 on 0.25 = %v, want 0.0625", values[0])
	}
}

func TestBackgroundCompositesTransparency(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 2, 1))
	src.SetNRGBA(0, 0, color.NRGBA{255, 255, 255, 0})
	src.SetNRGBA(1, 0, color.NRGBA{255, 255, 255, 255})

	black := FromImageWithBackground(src, "black")
	white := FromImageWithBackground(src, "white")

	if black.L[0] != 0 {
		t.Fatalf("transparent pixel over black = %v, want 0", black.L[0])
	}
	if white.L[0] != 1 {
		t.Fatalf("transparent pixel over white = %v, want 1", white.L[0])
	}
	if black.L[1] != white.L[1] {
		t.Fatalf("opaque pixel changed with background: %v vs %v", black.L[1], white.L[1])
	}

	auto := FromImageWithBackground(src, "auto")
	if auto.L[0] != 0 {
		t.Fatalf("auto background should choose black for a light subject: %v", auto.L[0])
	}

	dark := image.NewNRGBA(image.Rect(0, 0, 2, 1))
	dark.SetNRGBA(0, 0, color.NRGBA{255, 255, 255, 0})
	dark.SetNRGBA(1, 0, color.NRGBA{0, 0, 0, 255})
	darkAuto := FromImageWithBackground(dark, "auto")
	if darkAuto.L[0] != 1 {
		t.Fatalf("auto background should choose white for a dark subject: %v", darkAuto.L[0])
	}
}

func TestResampleAveragesAlpha(t *testing.T) {
	img := &LinearImage{
		Width: 2, Height: 1,
		R: []float32{0.2, 0.8},
		G: []float32{0.2, 0.8},
		B: []float32{0.2, 0.8},
		L: []float32{0.2, 0.8},
		A: []float32{0, 1},
	}
	grid := Resample(img, 2, 1)
	if grid.A[0] != 0 || grid.A[1] != 1 {
		t.Fatalf("alpha = %v,%v, want 0,1", grid.A[0], grid.A[1])
	}
}

func TestResampleDefaultsToOpaqueWithoutAlpha(t *testing.T) {
	img := &LinearImage{
		Width: 1, Height: 1,
		R: []float32{0.5}, G: []float32{0.5}, B: []float32{0.5}, L: []float32{0.5},
	}
	grid := Resample(img, 1, 1)
	if grid.A[0] != 1 {
		t.Fatalf("alpha without source data = %v, want 1", grid.A[0])
	}
}
