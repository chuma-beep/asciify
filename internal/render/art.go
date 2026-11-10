// Package render turns normalized grids into character art and output formats.
package render

import (
	"math"
	"runtime"
	"sync"

	"asciify/internal/glyphs"
	"asciify/internal/palette"
	"asciify/internal/pipeline"
)

// Color is an 8-bit sRGB colour.
type Color struct{ R, G, B uint8 }

// Art is a grid of characters with optional foreground/background colours.
type Art struct {
	Cols, Rows int
	Chars      []rune
	FG, BG     []Color
	Halfblock  bool
	Monochrome bool

	// Paint output. FillMask marks cells that draw a solid background; BG is
	// the fill and FG is a contrasting ink. TextChars is a tone-mapped glyph
	// fallback for plain text, which cannot draw backgrounds. Base is the
	// resolved canvas colour behind transparent images.
	FillMask  []bool
	TextChars []rune
	Base      Color
}

func colorFromLinear(r, g, b float32) Color {
	return Color{
		R: uint8(math.Round(pipeline.LinearToSRGB(float64(r)) * 255)),
		G: uint8(math.Round(pipeline.LinearToSRGB(float64(g)) * 255)),
		B: uint8(math.Round(pipeline.LinearToSRGB(float64(b)) * 255)),
	}
}

// Ramp maps luminance to the coverage-sorted glyph ramp.
func Ramp(grid *pipeline.Grid, set *glyphs.Set, dither, invert bool) *Art {
	art := &Art{Cols: grid.Cols, Rows: grid.Rows,
		Chars: make([]rune, grid.Cols*grid.Rows),
		FG:    make([]Color, grid.Cols*grid.Rows),
	}
	ramp := set.Ramp()
	if len(ramp) == 0 {
		return art
	}
	errs := make([]float32, len(grid.L))
	for i := 0; i < len(grid.L); i++ {
		v := grid.L[i]
		if invert {
			v = 1 - v
		}
		if dither {
			v += errs[i]
			if v < 0 {
				v = 0
			} else if v > 1 {
				v = 1
			}
		}
		best := 0
		bestDist := float32(1e30)
		for gi := range ramp {
			dist := float32(math.Abs(float64(v - ramp[gi].Coverage)))
			if dist < bestDist {
				bestDist = dist
				best = gi
			}
		}
		art.Chars[i] = ramp[best].Rune
		art.FG[i] = colorFromLinear(grid.R[i], grid.G[i], grid.B[i])
		if dither {
			quantized := ramp[best].Coverage
			err := v - quantized
			x, y := i%grid.Cols, i/grid.Cols
			addErr(errs, grid.Cols, grid.Rows, x+1, y, err*7/16)
			addErr(errs, grid.Cols, grid.Rows, x-1, y+1, err*3/16)
			addErr(errs, grid.Cols, grid.Rows, x, y+1, err*5/16)
			addErr(errs, grid.Cols, grid.Rows, x+1, y+1, err/16)
		}
	}
	return art
}

func addErr(errs []float32, cols, rows, x, y int, value float32) {
	if x < 0 || x >= cols || y < 0 || y >= rows {
		return
	}
	errs[y*cols+x] += value
}

// Shape chooses the glyph whose raster best matches each image tile.
func Shape(grid, tiles *pipeline.Grid, set *glyphs.Set, invert bool) *Art {
	art := &Art{Cols: grid.Cols, Rows: grid.Rows,
		Chars: make([]rune, grid.Cols*grid.Rows),
		FG:    make([]Color, grid.Cols*grid.Rows),
	}
	cellW, cellH := set.CellW, set.CellH
	rows := grid.Rows
	procs := runtime.GOMAXPROCS(0)
	if procs > rows {
		procs = rows
	}
	var wg sync.WaitGroup
	for p := 0; p < procs; p++ {
		wg.Add(1)
		go func(start int) {
			defer wg.Done()
			tile := make([]float32, cellW*cellH)
			blurred := make([]float32, cellW*cellH)
			tmp := make([]float32, cellW*cellH)
			for y := start; y < rows; y += procs {
				for x := 0; x < grid.Cols; x++ {
					for ty := 0; ty < cellH; ty++ {
						src := (y*cellH+ty)*tiles.Cols + x*cellW
						copy(tile[ty*cellW:(ty+1)*cellW], tiles.L[src:src+cellW])
					}
					blurTile(tile, blurred, tmp, cellW, cellH)
					best := set.Best(blurred, invert)
					i := y*grid.Cols + x
					if best != nil {
						art.Chars[i] = best.Rune
					} else {
						art.Chars[i] = ' '
					}
					art.FG[i] = colorFromLinear(grid.R[i], grid.G[i], grid.B[i])
				}
			}
		}(p)
	}
	wg.Wait()
	return art
}

// blurTile applies a light 3x3 box blur so sensor grain does not dominate
// shape matching. src and dst must be cellW*cellH; tmp is scratch space.
func blurTile(src, dst, tmp []float32, w, h int) {
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var sum float32
			for dy := -1; dy <= 1; dy++ {
				yy := y + dy
				if yy < 0 {
					yy = 0
				} else if yy >= h {
					yy = h - 1
				}
				for dx := -1; dx <= 1; dx++ {
					xx := x + dx
					if xx < 0 {
						xx = 0
					} else if xx >= w {
						xx = w - 1
					}
					sum += src[yy*w+xx]
				}
			}
			tmp[y*w+x] = sum / 9
		}
	}
	copy(dst, tmp)
}

// Halfblock builds a double-height art using the upper half-block character.
// grid must be sampled at cols x (characterRows*2).
func Halfblock(grid *pipeline.Grid, withColor bool) *Art {
	rows := grid.Rows / 2
	art := &Art{Cols: grid.Cols, Rows: rows, Halfblock: true,
		Chars: make([]rune, grid.Cols*rows),
		FG:    make([]Color, grid.Cols*rows),
		BG:    make([]Color, grid.Cols*rows),
	}
	for y := 0; y < rows; y++ {
		for x := 0; x < grid.Cols; x++ {
			i := y*grid.Cols + x
			art.Chars[i] = '▀'
			if withColor {
				top := (y * 2 * grid.Cols) + x
				bottom := ((y*2 + 1) * grid.Cols) + x
				art.FG[i] = colorFromLinear(grid.R[top], grid.G[top], grid.B[top])
				art.BG[i] = colorFromLinear(grid.R[bottom], grid.G[bottom], grid.B[bottom])
			}
		}
	}
	return art
}

type paintRGB struct{ r, g, b float32 }

func linearRGB(c Color) paintRGB {
	return paintRGB{
		r: float32(pipeline.SRGBToLinear(float64(c.R) / 255)),
		g: float32(pipeline.SRGBToLinear(float64(c.G) / 255)),
		b: float32(pipeline.SRGBToLinear(float64(c.B) / 255)),
	}
}

func (c paintRGB) color() Color {
	return colorFromLinear(c.r, c.g, c.b)
}

func mixPaint(a, b paintRGB, t float32) paintRGB {
	if t < 0 {
		t = 0
	} else if t > 1 {
		t = 1
	}
	return paintRGB{
		r: a.r + (b.r-a.r)*t,
		g: a.g + (b.g-a.g)*t,
		b: a.b + (b.b-a.b)*t,
	}
}

// cellHash returns a deterministic value in [0,1) for a cell and seed.
func cellHash(x, y int, seed int64) float32 {
	h := uint64(uint32(x))*0x9e3779b97f4a7c15 ^
		uint64(uint32(y))*0xc2b2ae3d27d4eb4f ^
		uint64(seed)
	h ^= h >> 29
	h *= 0xbf58476d1ce4e5b9
	h ^= h >> 32
	return float32(h>>40) / float32(1<<24)
}

// rampGlyph returns the glyph whose coverage is closest to target. Texture
// lookups skip the space glyph so a filled cell always carries a mark.
func rampGlyph(ramp []glyphs.Glyph, target float32, allowSpace bool) *glyphs.Glyph {
	var best *glyphs.Glyph
	bestDist := float32(1e30)
	for i := range ramp {
		if !allowSpace && ramp[i].Rune == ' ' {
			continue
		}
		dist := float32(math.Abs(float64(ramp[i].Coverage - target)))
		if dist < bestDist {
			bestDist = dist
			best = &ramp[i]
		}
	}
	return best
}

// unpremultiply recovers a subject colour from a cell that was composited over
// base with opacity a.
func unpremultiply(cell, base paintRGB, a float32) paintRGB {
	if a <= 1e-4 {
		return base
	}
	inv := 1 - a
	return paintRGB{
		r: clamp01((cell.r - base.r*inv) / a),
		g: clamp01((cell.g - base.g*inv) / a),
		b: clamp01((cell.b - base.b*inv) / a),
	}
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

// detailMap measures local luminance variation, then thresholds and blurs it
// so texture decisions ignore single-pixel noise and follow actual structure.
func detailMap(grid *pipeline.Grid) []float32 {
	n := grid.Cols * grid.Rows
	raw := make([]float32, n)
	for y := 0; y < grid.Rows; y++ {
		for x := 0; x < grid.Cols; x++ {
			i := y*grid.Cols + x
			var d float32
			if x > 0 && x < grid.Cols-1 {
				d += float32(math.Abs(float64(grid.L[i+1]-grid.L[i-1]))) / 2
			}
			if y > 0 && y < grid.Rows-1 {
				d += float32(math.Abs(float64(grid.L[i+grid.Cols]-grid.L[i-grid.Cols]))) / 2
			}
			raw[i] = d
		}
	}

	out := make([]float32, n)
	for y := 0; y < grid.Rows; y++ {
		for x := 0; x < grid.Cols; x++ {
			var sum float32
			var count int
			for dy := -1; dy <= 1; dy++ {
				yy := y + dy
				if yy < 0 || yy >= grid.Rows {
					continue
				}
				for dx := -1; dx <= 1; dx++ {
					xx := x + dx
					if xx < 0 || xx >= grid.Cols {
						continue
					}
					sum += raw[yy*grid.Cols+xx]
					count++
				}
			}
			out[y*grid.Cols+x] = clamp01((sum/float32(count) - 0.03) * 6)
		}
	}
	return out
}

// textureRamp drops structural glyphs that read as rain or scaffolding when
// scattered over a photographic surface.
func textureRamp(ramp []glyphs.Glyph) []glyphs.Glyph {
	out := make([]glyphs.Glyph, 0, len(ramp))
	for _, g := range ramp {
		if g.Rune == ' ' || g.Coverage > 0.16 || isStructural(g.Rune) {
			continue
		}
		out = append(out, g)
	}
	if len(out) == 0 {
		for _, g := range ramp {
			if g.Rune != ' ' {
				out = append(out, g)
			}
		}
	}
	return out
}

func isStructural(r rune) bool {
	switch r {
	case '|', '/', '\\', '(', ')', '[', ']', '{', '}', '<', '>', '!', 'i', 'l', 'I', '1':
		return true
	}
	return false
}

// Paint implements the scene-style cell treatment used by the hand-authored
// ASCII scenes. Each covered cell draws a solid fill in its average colour,
// then a glyph in a contrasting ink so the glyph reads as texture rather than
// as the only tone channel. Partial cells draw only a sparse edge glyph, which
// keeps silhouettes crisp and leaves true negative space.
func Paint(grid *pipeline.Grid, set *glyphs.Set, texture float32, seed int64, invert bool, base Color, pal *palette.Palette, dither bool) *Art {
	n := grid.Cols * grid.Rows
	art := &Art{
		Cols: grid.Cols, Rows: grid.Rows,
		Chars:     make([]rune, n),
		FG:        make([]Color, n),
		BG:        make([]Color, n),
		FillMask:  make([]bool, n),
		TextChars: make([]rune, n),
		Base:      base,
	}
	ramp := set.Ramp()
	if len(ramp) == 0 {
		return art
	}
	if texture < 0 {
		texture = 0
	} else if texture > 1 {
		texture = 1
	}
	inkMix := 0.05 + 0.28*texture
	baseLin := linearRGB(base)
	details := detailMap(grid)
	softRamp := textureRamp(ramp)

	for y := 0; y < grid.Rows; y++ {
		for x := 0; x < grid.Cols; x++ {
			i := y*grid.Cols + x
			cov := float32(1)
			if grid.A != nil && i < len(grid.A) {
				cov = clamp01(grid.A[i])
			}
			fillLin := paintRGB{grid.R[i], grid.G[i], grid.B[i]}
			fill := fillLin.color()
			l := clamp01(grid.L[i])
			if pal != nil {
				// Curated looks act as a tonal ramp: the fill follows the
				// palette's light-to-dark stops instead of snapping to every
				// accent, which keeps photographs coherent.
				tone := float64(l)
				if dither {
					quantized := pal.ToneDither(tone, x, y)
					fill = Color{R: quantized.R, G: quantized.G, B: quantized.B}
				} else {
					quantized := pal.Tone(tone)
					fill = Color{R: quantized.R, G: quantized.G, B: quantized.B}
				}
				fillLin = linearRGB(fill)
			}
			detail := details[i]

			art.Chars[i] = ' '
			art.FG[i] = fill
			art.BG[i] = fill

			// Plain-text output cannot draw backgrounds, so it always uses a
			// tone-mapped glyph. This is the quality floor for copy/paste.
			tone := l
			if invert {
				tone = 1 - tone
			}
			if g := rampGlyph(ramp, tone, true); g != nil {
				art.TextChars[i] = g.Rune
			} else {
				art.TextChars[i] = ' '
			}

			if cov < 0.12 {
				continue
			}

			flick := cellHash(x, y, seed)
			if cov < 0.5 {
				// Feather partial cells with a sparse edge mark in the
				// subject's recovered colour, never a blocky half-fill.
				subject := unpremultiply(fillLin, baseLin, cov)
				if pal != nil {
					tone := pal.Tone(float64(l))
					subject = linearRGB(Color{R: tone.R, G: tone.G, B: tone.B})
				}
				edgeColor := mixPaint(baseLin, subject, clamp01(cov*1.6)).color()
				art.FG[i] = edgeColor
				t := 0.025 + 0.12*cov + (flick-0.5)*0.03
				if g := rampGlyph(ramp, clamp01(t), false); g != nil {
					art.Chars[i] = g.Rune
				}
				continue
			}

			art.FillMask[i] = true
			art.BG[i] = fill
			// A flat region is mostly pure fill. Glyphs appear mainly where
			// the image has structure, so large backgrounds stay quiet.
			if flick > 0.03+0.75*detail {
				continue
			}
			// Dense marks on dark fills, sparse marks on bright fills, with a
			// little deterministic flicker to break flat bands.
			t := 0.015 + 0.07*(1-l) + 0.07*detail + (cellHash(x, y, seed+1)-0.5)*0.03
			if g := rampGlyph(softRamp, clamp01(t), false); g != nil {
				art.Chars[i] = g.Rune
			} else {
				art.Chars[i] = '.'
			}
			target := paintRGB{1, 1, 1}
			if l >= 0.5 {
				target = paintRGB{0, 0, 0}
			}
			strength := inkMix * (0.4 + 0.6*detail)
			art.FG[i] = mixPaint(fillLin, target, strength).color()
		}
	}
	return art
}
