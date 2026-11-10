package pipeline

import (
	"math"
	"runtime"
	"sync"
)

// CellRows returns the character rows needed for cols columns and a terminal
// cell that is cellAspect times taller than it is wide.
func CellRows(srcW, srcH, cols int, cellAspect float64) int {
	if srcW <= 0 || srcH <= 0 || cols <= 0 {
		return 1
	}
	rows := int(math.Round(float64(srcH) / float64(srcW) * float64(cols) / cellAspect))
	if rows < 1 {
		return 1
	}
	return rows
}

type span struct {
	start, end int
}

// Resample area-averages the linear image into cols x rows. It never uses
// nearest-neighbour sampling. Output rows are split across goroutines.
func Resample(img *LinearImage, cols, rows int) *Grid {
	if cols < 1 {
		cols = 1
	}
	if rows < 1 {
		rows = 1
	}
	grid := &Grid{
		Cols: cols, Rows: rows,
		L: make([]float32, cols*rows),
		R: make([]float32, cols*rows),
		G: make([]float32, cols*rows),
		B: make([]float32, cols*rows),
		A: make([]float32, cols*rows),
	}
	if img.Width <= 0 || img.Height <= 0 {
		return grid
	}
	for i := range grid.A {
		grid.A[i] = 1
	}
	hasAlpha := len(img.A) == len(img.R)

	xSpans := make([]span, cols)
	for ox := 0; ox < cols; ox++ {
		x0 := float64(ox) * float64(img.Width) / float64(cols)
		x1 := float64(ox+1) * float64(img.Width) / float64(cols)
		xSpans[ox] = span{start: int(math.Floor(x0)), end: int(math.Ceil(x1))}
	}

	workers := runtime.GOMAXPROCS(0)
	if workers > rows {
		workers = rows
	}
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			for oy := worker; oy < rows; oy += workers {
				y0 := float64(oy) * float64(img.Height) / float64(rows)
				y1 := float64(oy+1) * float64(img.Height) / float64(rows)
				syStart := int(math.Floor(y0))
				syEnd := int(math.Ceil(y1))
				for ox := 0; ox < cols; ox++ {
					x0 := float64(ox) * float64(img.Width) / float64(cols)
					x1 := float64(ox+1) * float64(img.Width) / float64(cols)
					var sr, sg, sb, sa, area float64
					for sy := syStart; sy < syEnd; sy++ {
						wy := math.Min(y1, float64(sy+1)) - math.Max(y0, float64(sy))
						if wy <= 0 {
							continue
						}
						row := sy * img.Width
						for sx := xSpans[ox].start; sx < xSpans[ox].end; sx++ {
							wx := math.Min(x1, float64(sx+1)) - math.Max(x0, float64(sx))
							if wx <= 0 {
								continue
							}
							weight := wx * wy
							i := row + sx
							sr += float64(img.R[i]) * weight
							sg += float64(img.G[i]) * weight
							sb += float64(img.B[i]) * weight
							if hasAlpha {
								sa += float64(img.A[i]) * weight
							}
							area += weight
						}
					}
					if area == 0 {
						continue
					}
					r := float32(sr / area)
					g := float32(sg / area)
					b := float32(sb / area)
					i := oy*cols + ox
					grid.R[i] = r
					grid.G[i] = g
					grid.B[i] = b
					grid.L[i] = Luminance(r, g, b)
					if hasAlpha {
						grid.A[i] = float32(sa / area)
					}
				}
			}
		}(w)
	}
	wg.Wait()
	return grid
}
