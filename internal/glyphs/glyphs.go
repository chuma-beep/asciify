// Package glyphs rasterizes candidate characters from a monospace font and
// ranks them by measured ink coverage.
package glyphs

import (
	_ "embed"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"
	"sort"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

//go:embed assets/Go-Mono.ttf
var defaultFont []byte

// DefaultCharset is the candidate set used when --charset is not supplied.
const DefaultCharset = ".,:;'\"`-_~=+*#%@/\\|()[]{}<>!1lIiO0"

// Glyph is one rasterized character.
type Glyph struct {
	Rune     rune
	Bitmap   []float32
	Coverage float32
	Sum      float32
	SumSq    float32
}

// Set is a rasterized, coverage-sorted glyph alphabet.
type Set struct {
	Glyphs   []Glyph
	CellW    int
	CellH    int
	FontName string
}

// DefaultFont returns the embedded Go Mono TTF.
func DefaultFont() []byte { return defaultFont }

// Load parses a TTF and builds a coverage-sorted glyph set.
func Load(fontData []byte, charset string, cellW, cellH int) (*Set, error) {
	if cellW < 1 || cellH < 1 {
		return nil, fmt.Errorf("cell size must be positive")
	}
	parsed, err := opentype.Parse(fontData)
	if err != nil {
		return nil, fmt.Errorf("parse font: %w", err)
	}
	face, err := opentype.NewFace(parsed, &opentype.FaceOptions{
		Size:    float64(cellH) * 0.86,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return nil, fmt.Errorf("create face: %w", err)
	}
	defer face.Close()

	if charset == "" {
		charset = DefaultCharset
	}
	seen := make(map[rune]bool)
	runes := make([]rune, 0, len(charset)+1)
	for _, r := range " " + charset {
		if seen[r] {
			continue
		}
		seen[r] = true
		runes = append(runes, r)
	}

	metrics := face.Metrics()
	ascent := metrics.Ascent.Ceil()
	descent := metrics.Descent.Ceil()
	textHeight := ascent + descent
	baseline := (cellH-textHeight)/2 + ascent

	set := &Set{CellW: cellW, CellH: cellH, FontName: "Go Mono"}
	for _, r := range runes {
		dst := image.NewAlpha(image.Rect(0, 0, cellW, cellH))
		advance, ok := face.GlyphAdvance(r)
		if !ok {
			continue
		}
		dotX := (cellW - advance.Ceil()) / 2
		if dotX < 0 {
			dotX = 0
		}
		d := font.Drawer{
			Dst:  dst,
			Src:  image.NewUniform(color.Alpha{255}),
			Face: face,
			Dot:  fixed.P(dotX, baseline),
		}
		d.DrawString(string(r))

		bitmap := make([]float32, cellW*cellH)
		var sum float32
		for i, a := range dst.Pix {
			v := float32(a) / 255
			bitmap[i] = v
			sum += v
		}
		coverage := sum / float32(len(bitmap))
		var sumSq float32
		for _, v := range bitmap {
			sumSq += v * v
		}
		set.Glyphs = append(set.Glyphs, Glyph{
			Rune: r, Bitmap: bitmap, Coverage: coverage, Sum: sum, SumSq: sumSq,
		})
	}
	sort.SliceStable(set.Glyphs, func(i, j int) bool {
		return set.Glyphs[i].Coverage < set.Glyphs[j].Coverage
	})
	return set, nil
}

// Ramp returns the coverage-sorted glyphs.
func (s *Set) Ramp() []Glyph { return s.Glyphs }

// Best returns the glyph minimizing mean-squared error against tile.
// tile must have CellW*CellH entries. The loop is allocation-free.
func (s *Set) Best(tile []float32, invert bool) *Glyph {
	if len(s.Glyphs) == 0 {
		return nil
	}
	n := s.CellW * s.CellH
	if len(tile) < n {
		return nil
	}

	var tileSum, tileSumSq float32
	for i := 0; i < n; i++ {
		v := tile[i]
		tileSum += v
		tileSumSq += v * v
	}
	rawMean := tileSum / float32(n)
	rawVar := tileSumSq/float32(n) - rawMean*rawMean
	if rawVar < 0 {
		rawVar = 0
	}
	std := float32(math.Sqrt(float64(rawVar)))

	toneMean := rawMean
	if invert {
		toneMean = 1 - toneMean
	}

	const coverageWeight = 0.1
	coverageScale := coverageWeight * float32(n)

	// A flat tile has no shape to match. Keep its tone and let the ramp
	// choose a glyph by coverage instead of forcing an arbitrary edge.
	if std < 0.12 {
		best := &s.Glyphs[0]
		bestDist := float32(1e30)
		for gi := range s.Glyphs {
			delta := float32(math.Abs(float64(s.Glyphs[gi].Coverage - toneMean)))
			if delta < bestDist {
				bestDist = delta
				best = &s.Glyphs[gi]
			}
		}
		return best
	}

	// Normalize local structure around the tile's mean so faint face and
	// foliage detail still contributes to the MSE. The coverage prior uses
	// the original tone, so the glyph still carries the cell brightness.
	gain := 0.12 / std
	if gain < 1 {
		gain = 1
	} else if gain > 4 {
		gain = 4
	}
	const target = float32(0.5)
	normalizedSumSq := target*target*float32(n) + gain*gain*rawVar*float32(n)

	best := &s.Glyphs[0]
	bestScore := float32(1e30)
	for gi := range s.Glyphs {
		g := &s.Glyphs[gi]
		var dot float32
		for i := 0; i < n; i++ {
			dot += tile[i] * g.Bitmap[i]
		}
		if invert {
			dot = g.Sum - dot
		}
		// dot is against the raw tile; transform to the locally normalized tile.
		normalizedDot := target*g.Sum + gain*(dot-rawMean*g.Sum)
		score := normalizedSumSq + g.SumSq - 2*normalizedDot
		delta := g.Coverage - toneMean
		score += coverageScale * delta * delta
		if score < bestScore {
			bestScore = score
			best = g
		}
	}
	return best
}

// DrawRune is a small helper used by the PNG output stage.
func DrawRune(dst draw.Image, face font.Face, r rune, x, y int, src image.Image) {
	d := font.Drawer{Dst: dst, Src: src, Face: face, Dot: fixed.P(x, y)}
	d.DrawString(string(r))
}
