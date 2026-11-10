// Package convert turns image bytes into ASCII art.
//
// It is the shared core behind the command line tool and the browser build.
// Callers supply the image bytes and a complete Options value; the result is a
// render.Art that can be written as plain text, ANSI text, or a PNG.
package convert

import (
	"bytes"
	"errors"
	"fmt"

	"asciify/internal/glyphs"
	"asciify/internal/pipeline"
	"asciify/internal/render"
)

// DefaultCellW and DefaultCellH are the glyph cell dimensions used for shape
// matching and PNG output.
const (
	DefaultCellW = 8
	DefaultCellH = 16
)

// Options controls one conversion. The zero value is not usable; start from
// DefaultOptions and override the fields you need.
type Options struct {
	Cols          int
	Mode          string // "ramp" or "shape".
	ColorMode     string // "none", "ansi256", or "truecolor".
	Charset       string
	CellAspect    float64
	ClipLow       float64
	ClipHigh      float64
	Edges         bool
	LocalContrast bool
	Dither        bool
	Invert        bool
	Halfblock     bool
	// FontData is an optional monospace TTF. Nil uses the embedded Go Mono.
	FontData []byte
}

// DefaultOptions returns the same defaults as the CLI.
func DefaultOptions() Options {
	return Options{
		Cols:       120,
		Mode:       "shape",
		ColorMode:  "none",
		Charset:    glyphs.DefaultCharset,
		CellAspect: 2.0,
		ClipLow:    0.02,
		ClipHigh:   0.98,
	}
}

// Validate reports whether the options can be used for a conversion.
func (o Options) Validate() error {
	if o.Cols < 4 {
		return errors.New("cols must be at least 4")
	}
	if o.Mode != "ramp" && o.Mode != "shape" {
		return fmt.Errorf("unknown mode %q: use ramp or shape", o.Mode)
	}
	if o.ColorMode != "none" && o.ColorMode != "ansi256" && o.ColorMode != "truecolor" {
		return fmt.Errorf("unknown colour mode %q: use none, ansi256, or truecolor", o.ColorMode)
	}
	if o.CellAspect <= 0 {
		return errors.New("cell aspect must be positive")
	}
	if o.ClipLow < 0 || o.ClipHigh > 1 || o.ClipLow >= o.ClipHigh {
		return errors.New("clip low and high must satisfy 0 <= low < high <= 1")
	}
	return nil
}

// ResolvedFont returns the configured font, or the embedded Go Mono font.
func (o Options) ResolvedFont() []byte {
	if len(o.FontData) > 0 {
		return o.FontData
	}
	return glyphs.DefaultFont()
}

// Convert decodes imageData and renders it as ASCII art.
func Convert(imageData []byte, opts Options) (*render.Art, error) {
	if err := opts.Validate(); err != nil {
		return nil, err
	}
	img, err := pipeline.Decode(bytes.NewReader(imageData))
	if err != nil {
		return nil, err
	}
	return ConvertImage(img, opts)
}

// ConvertImage renders an already-decoded linear image.
func ConvertImage(img *pipeline.LinearImage, opts Options) (*render.Art, error) {
	if err := opts.Validate(); err != nil {
		return nil, err
	}

	rows := pipeline.CellRows(img.Width, img.Height, opts.Cols, opts.CellAspect)
	sampleRows := rows
	if opts.Halfblock {
		sampleRows = rows * 2
	}
	grid := pipeline.Resample(img, opts.Cols, sampleRows)

	low, high := pipeline.ClipPercentiles(grid, opts.ClipLow, opts.ClipHigh)
	pipeline.Normalize(grid, low, high)

	var (
		tiles *pipeline.Grid
		set   *glyphs.Set
		err   error
	)
	if !opts.Halfblock {
		set, err = glyphs.Load(opts.ResolvedFont(), opts.Charset, DefaultCellW, DefaultCellH)
		if err != nil {
			return nil, err
		}
		if opts.Mode == "shape" {
			tiles = pipeline.Resample(img, opts.Cols*DefaultCellW, sampleRows*DefaultCellH)
			pipeline.NormalizeValues(tiles.L, low, high)
		}
	}

	if opts.LocalContrast {
		pipeline.LocalContrast(grid, 3, 1.8)
		if tiles != nil {
			pipeline.LocalContrast(tiles, 3, 1.8)
		}
	}
	if opts.Edges {
		pipeline.Unsharp(grid, 1.2)
		if tiles != nil {
			pipeline.Unsharp(tiles, 1.2)
		}
	}

	var art *render.Art
	switch {
	case opts.Halfblock:
		art = render.Halfblock(grid, opts.ColorMode != "none")
	case opts.Mode == "shape":
		art = render.Shape(grid, tiles, set, opts.Invert)
	default:
		art = render.Ramp(grid, set, opts.Dither, opts.Invert)
	}
	art.Monochrome = opts.ColorMode == "none"
	return art, nil
}

// Text renders the art as plain text with trailing spaces trimmed.
func Text(art *render.Art) string {
	var b bytes.Buffer
	_ = render.WriteText(&b, art)
	return b.String()
}

// Terminal renders the art with ANSI escapes for the requested colour mode.
func Terminal(art *render.Art, colorMode string) string {
	var b bytes.Buffer
	_ = render.WriteTerminal(&b, art, colorMode)
	return b.String()
}

// PNG renders the art as a PNG image using the supplied font.
func PNG(art *render.Art, fontData []byte) ([]byte, error) {
	var b bytes.Buffer
	if err := render.WritePNG(&b, art, fontData, DefaultCellW, DefaultCellH); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}
