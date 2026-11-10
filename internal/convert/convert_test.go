package convert

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"asciify/internal/glyphs"
	"asciify/internal/palette"
)

func TestConvertGoldenRamp(t *testing.T) {
	image, err := os.ReadFile(filepath.Join("..", "..", "testdata", "golden.png"))
	if err != nil {
		t.Fatal(err)
	}

	opts := DefaultOptions()
	opts.Cols = 16
	opts.Mode = "ramp"
	art, err := Convert(image, opts)
	if err != nil {
		t.Fatal(err)
	}

	want := "     @@@@@@00000\n     @@@@@@00000\n     @@@@@@00000\n     @@@@@@00000\n"
	if got := Text(art); got != want {
		t.Fatalf("golden mismatch:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestConvertRejectsInvalidOptions(t *testing.T) {
	opts := DefaultOptions()
	opts.Cols = 2
	if _, err := Convert([]byte{}, opts); err == nil {
		t.Fatal("expected an error for cols below 4")
	}

	opts = DefaultOptions()
	opts.Mode = "painted"
	if _, err := Convert([]byte{}, opts); err == nil {
		t.Fatal("expected an error for an unknown mode")
	}

	opts = DefaultOptions()
	opts.Gamma = 0
	if _, err := Convert([]byte{}, opts); err == nil {
		t.Fatal("expected an error for a non-positive gamma")
	}

	opts = DefaultOptions()
	opts.Background = "blue"
	if _, err := Convert([]byte{}, opts); err == nil {
		t.Fatal("expected an error for an unknown background")
	}
}

func TestGammaChangesOutput(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "gradient.png"))
	if err != nil {
		t.Fatal(err)
	}

	base := DefaultOptions()
	base.Cols = 32
	base.Mode = "ramp"

	dark := base
	dark.Gamma = 0.6
	light := base
	light.Gamma = 1.8

	darkArt, err := Convert(data, dark)
	if err != nil {
		t.Fatal(err)
	}
	lightArt, err := Convert(data, light)
	if err != nil {
		t.Fatal(err)
	}
	if Text(darkArt) == Text(lightArt) {
		t.Fatal("gamma did not change the output")
	}
}

func TestBackgroundCompositesTransparency(t *testing.T) {
	source := image.NewNRGBA(image.Rect(0, 0, 48, 24))
	for y := 0; y < 24; y++ {
		for x := 0; x < 24; x++ {
			source.SetNRGBA(x, y, color.NRGBA{255, 255, 255, 255})
		}
	}
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, source); err != nil {
		t.Fatal(err)
	}

	opts := DefaultOptions()
	opts.Cols = 16
	opts.Mode = "ramp"
	opts.Background = "black"
	blackArt, err := Convert(encoded.Bytes(), opts)
	if err != nil {
		t.Fatal(err)
	}
	opts.Background = "white"
	whiteArt, err := Convert(encoded.Bytes(), opts)
	if err != nil {
		t.Fatal(err)
	}
	if Text(blackArt) == Text(whiteArt) {
		t.Fatal("background choice did not affect transparent pixels")
	}
}

func TestHTMLOutput(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "circle.png"))
	if err != nil {
		t.Fatal(err)
	}
	opts := DefaultOptions()
	opts.Cols = 24
	opts.ColorMode = "truecolor"
	art, err := Convert(data, opts)
	if err != nil {
		t.Fatal(err)
	}
	html := HTML(art)
	if html == "" || !strings.Contains(html, "<span") {
		t.Fatalf("unexpected HTML output: %q", html)
	}
}

func TestPNGOutput(t *testing.T) {
	image, err := os.ReadFile(filepath.Join("..", "..", "testdata", "circle.png"))
	if err != nil {
		t.Fatal(err)
	}

	opts := DefaultOptions()
	opts.Cols = 24
	art, err := Convert(image, opts)
	if err != nil {
		t.Fatal(err)
	}
	png, err := PNG(art, glyphs.DefaultFont())
	if err != nil {
		t.Fatal(err)
	}
	if len(png) == 0 {
		t.Fatal("PNG output is empty")
	}
}

func TestPaintModeFillsCells(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "circle.png"))
	if err != nil {
		t.Fatal(err)
	}
	opts := DefaultOptions()
	opts.Cols = 24
	opts.Mode = "paint"
	opts.ColorMode = "truecolor"
	art, err := Convert(data, opts)
	if err != nil {
		t.Fatal(err)
	}
	if art.FillMask == nil {
		t.Fatal("paint mode did not produce fill data")
	}
	filled := 0
	for _, ok := range art.FillMask {
		if ok {
			filled++
		}
	}
	if filled == 0 {
		t.Fatal("paint mode filled no cells")
	}
	if html := HTML(art); !strings.Contains(html, "background:") {
		t.Fatalf("paint HTML has no cell backgrounds: %q", html[:min(120, len(html))])
	}
	if Text(art) == "" {
		t.Fatal("paint mode has no plain-text fallback")
	}
}

func TestPaintMonochromeUsesGreyFills(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "circle.png"))
	if err != nil {
		t.Fatal(err)
	}
	opts := DefaultOptions()
	opts.Cols = 12
	opts.Mode = "paint"
	opts.ColorMode = "none"
	art, err := Convert(data, opts)
	if err != nil {
		t.Fatal(err)
	}
	for i, ok := range art.FillMask {
		if !ok {
			continue
		}
		if art.BG[i].R != art.BG[i].G || art.BG[i].G != art.BG[i].B {
			t.Fatalf("monochrome paint fill is not grey: %+v", art.BG[i])
		}
	}
}

func TestValidateRejectsBadTexture(t *testing.T) {
	opts := DefaultOptions()
	opts.Texture = 1.5
	if err := opts.Validate(); err == nil {
		t.Fatal("expected an error for texture above 1")
	}
	opts.Texture = -0.1
	if err := opts.Validate(); err == nil {
		t.Fatal("expected an error for negative texture")
	}
}

func TestValidateRejectsUnknownPalette(t *testing.T) {
	opts := DefaultOptions()
	opts.Palette = "sunset"
	if err := opts.Validate(); err == nil {
		t.Fatal("expected an error for an unknown palette")
	}
}

func TestPaintPaletteQuantizesFills(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "photo.png"))
	if err != nil {
		t.Fatal(err)
	}
	pal, ok := palette.Get("abyss")
	if !ok {
		t.Fatal("abyss palette missing")
	}

	opts := DefaultOptions()
	opts.Cols = 32
	opts.Mode = "paint"
	opts.ColorMode = "truecolor"
	opts.Palette = "abyss"
	art, err := Convert(data, opts)
	if err != nil {
		t.Fatal(err)
	}
	filled := 0
	for _, ok := range art.FillMask {
		if ok {
			filled++
		}
	}
	if filled == 0 {
		t.Fatal("paint with a palette produced no fills")
	}
	if art.Base.R != pal.Background.R || art.Base.G != pal.Background.G || art.Base.B != pal.Background.B {
		t.Fatalf("paint base = %+v, want palette background %+v", art.Base, pal.Background)
	}
}

func TestShapePaletteQuantizesGlyphs(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "circle.png"))
	if err != nil {
		t.Fatal(err)
	}
	pal, ok := palette.Get("mono")
	if !ok {
		t.Fatal("mono palette missing")
	}
	allowed := make(map[[3]uint8]bool, len(pal.Colors))
	for _, c := range pal.Colors {
		allowed[[3]uint8{c.R, c.G, c.B}] = true
	}

	opts := DefaultOptions()
	opts.Cols = 24
	opts.Mode = "shape"
	opts.ColorMode = "truecolor"
	opts.Palette = "mono"
	art, err := Convert(data, opts)
	if err != nil {
		t.Fatal(err)
	}
	for i, ch := range art.Chars {
		if ch == ' ' {
			continue
		}
		if !allowed[[3]uint8{art.FG[i].R, art.FG[i].G, art.FG[i].B}] {
			t.Fatalf("glyph colour %+v is not in the mono palette", art.FG[i])
		}
	}
}
