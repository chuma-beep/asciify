package pipeline

import "math"

// ClipPercentiles finds the low/high luminance values at the requested
// percentiles. Percentiles are clamped to [0,1] and must be increasing.
func ClipPercentiles(grid *Grid, lowP, highP float64) (float32, float32) {
	if lowP < 0 {
		lowP = 0
	}
	if highP > 1 {
		highP = 1
	}
	if highP < lowP {
		lowP, highP = highP, lowP
	}
	const bins = 4096
	var hist [bins]int
	for _, v := range grid.L {
		idx := int(clamp01(v) * (bins - 1))
		hist[idx]++
	}
	total := len(grid.L)
	if total == 0 {
		return 0, 1
	}
	lowCount := int(float64(total) * lowP)
	highCount := int(float64(total) * highP)
	if highCount < lowCount {
		highCount = lowCount
	}
	var cumulative int
	var low, high float32
	for i, count := range hist {
		cumulative += count
		v := float32(i) / float32(bins-1)
		if cumulative > lowCount && low == 0 {
			low = v
		}
		if cumulative >= highCount {
			high = v
			break
		}
	}
	if high <= low {
		return 0, 1
	}
	return low, high
}

// Normalize stretches luminance from [low,high] into [0,1] in place.
func Normalize(grid *Grid, low, high float32) {
	if high <= low {
		return
	}
	rangeV := high - low
	for i, v := range grid.L {
		grid.L[i] = clamp01((v - low) / rangeV)
	}
}

// NormalizeValues applies the same clip to a flat luminance slice.
func NormalizeValues(values []float32, low, high float32) {
	if high <= low {
		return
	}
	rangeV := high - low
	for i, v := range values {
		values[i] = clamp01((v - low) / rangeV)
	}
}

// ApplyGamma maps normalized values through v^(1/gamma). Values above 1 lift
// midtones; values below 1 deepen them. Gamma 1 is a no-op.
func ApplyGamma(grid *Grid, gamma float64) {
	ApplyGammaValues(grid.L, gamma)
}

// ApplyGammaValues applies the same gamma curve to a flat luminance slice.
func ApplyGammaValues(values []float32, gamma float64) {
	if gamma <= 0 || gamma == 1 {
		return
	}
	exponent := 1 / gamma
	for i, v := range values {
		values[i] = clamp01(float32(math.Pow(float64(clamp01(v)), exponent)))
	}
}

// LocalContrast applies a box-blur based local contrast adjustment.
func LocalContrast(grid *Grid, radius int, strength float32) {
	if radius < 1 {
		radius = 1
	}
	blur := BoxBlur(grid.L, grid.Cols, grid.Rows, radius)
	for i, v := range grid.L {
		grid.L[i] = clamp01(blur[i] + (v-blur[i])*strength)
	}
}

// BoxBlur is a separable box blur over a flat row-major slice.
func BoxBlur(values []float32, cols, rows, radius int) []float32 {
	tmp := make([]float32, len(values))
	out := make([]float32, len(values))
	for y := 0; y < rows; y++ {
		row := y * cols
		var sum float64
		for x := -radius; x <= radius; x++ {
			if x >= 0 && x < cols {
				sum += float64(values[row+x])
			}
		}
		for x := 0; x < cols; x++ {
			tmp[row+x] = float32(sum / float64(2*radius+1))
			if x-radius >= 0 {
				sum -= float64(values[row+x-radius])
			}
			if x+radius+1 < cols {
				sum += float64(values[row+x+radius+1])
			}
		}
	}
	for x := 0; x < cols; x++ {
		var sum float64
		for y := -radius; y <= radius; y++ {
			if y >= 0 && y < rows {
				sum += float64(tmp[y*cols+x])
			}
		}
		for y := 0; y < rows; y++ {
			out[y*cols+x] = float32(sum / float64(2*radius+1))
			if y-radius >= 0 {
				sum -= float64(tmp[(y-radius)*cols+x])
			}
			if y+radius+1 < rows {
				sum += float64(tmp[(y+radius+1)*cols+x])
			}
		}
	}
	return out
}

// Unsharp applies an unsharp mask to luminance in place.
func Unsharp(grid *Grid, amount float32) {
	if amount <= 0 {
		return
	}
	blur := BoxBlur(grid.L, grid.Cols, grid.Rows, 2)
	for i, v := range grid.L {
		grid.L[i] = clamp01(v + (v-blur[i])*amount)
	}
}
