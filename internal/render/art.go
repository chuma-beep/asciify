// Package render turns normalized grids into character art and output formats.
package render

import (
	"math"
	"runtime"
	"sync"

	"asciify/internal/glyphs"
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
