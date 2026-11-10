// Package palette holds the curated colour looks used by the ASCII scenes.
//
// A palette is a small set of colours with a background and a few shades and
// accents. Image fills are matched in OKLab, so the result follows perceived
// lightness and hue rather than raw RGB distance.
package palette

import (
	"math"
	"sort"
)

// Color is an 8-bit sRGB colour.
type Color struct{ R, G, B uint8 }

// Palette is a named colour look. Colors always includes Background first.
type Palette struct {
	Name       string
	Dark       bool
	Background Color
	Colors     []Color

	labs     []lab
	ramp     []Color
	rampLabs []lab
}

type lab struct{ l, a, b float64 }

var order = []string{
	"abyss",
	"neon",
	"mono",
	"paper",
	"amber",
	"blueprint",
	"newsprint",
	"terminal",
}

var named = map[string]Palette{
	"abyss": newPalette("abyss", true,
		Color{0x00, 0x00, 0x00},
		Color{0x40, 0x43, 0x47},
		Color{0x61, 0x64, 0x6b},
		Color{0x85, 0x8a, 0x94},
		Color{0xaa, 0xb0, 0xbd},
		Color{0xcf, 0xd6, 0xe6},
		Color{0x33, 0xc8, 0xcc},
		Color{0xe0, 0x89, 0x2c},
	),
	"neon": newPalette("neon", true,
		Color{0x05, 0x07, 0x0c},
		Color{0x33, 0x4c, 0x48},
		Color{0x4a, 0x6f, 0x66},
		Color{0x65, 0x97, 0x88},
		Color{0x7f, 0xbf, 0xaa},
		Color{0x9a, 0xe6, 0xcd},
		Color{0x28, 0xd7, 0xff},
		Color{0xff, 0x3e, 0xa5},
		Color{0xf5, 0xff, 0x5a},
	),
	"mono": newPalette("mono", true,
		Color{0x0a, 0x0a, 0x0a},
		Color{0x49, 0x49, 0x49},
		Color{0x68, 0x68, 0x68},
		Color{0x8c, 0x8c, 0x8c},
		Color{0xb0, 0xb0, 0xb0},
		Color{0xd4, 0xd4, 0xd4},
		Color{0xff, 0xff, 0xff},
	),
	"paper": newPalette("paper", false,
		Color{0xf3, 0xee, 0xe4},
		Color{0xb8, 0xb8, 0xb3},
		Color{0x9b, 0x9d, 0x9b},
		Color{0x7a, 0x7d, 0x7f},
		Color{0x58, 0x5e, 0x63},
		Color{0x37, 0x3f, 0x47},
		Color{0xb2, 0x4a, 0x2e},
		Color{0x2f, 0x6f, 0x73},
	),
	"amber": newPalette("amber", true,
		Color{0x0a, 0x06, 0x03},
		Color{0x4f, 0x3a, 0x21},
		Color{0x71, 0x53, 0x30},
		Color{0x98, 0x71, 0x41},
		Color{0xbf, 0x8e, 0x52},
		Color{0xe6, 0xac, 0x63},
		Color{0xff, 0x8c, 0x42},
		Color{0xff, 0xd1, 0x66},
	),
	"blueprint": newPalette("blueprint", true,
		Color{0x04, 0x12, 0x1f},
		Color{0x2d, 0x48, 0x5e},
		Color{0x41, 0x63, 0x7d},
		Color{0x59, 0x81, 0xa1},
		Color{0x70, 0xa0, 0xc5},
		Color{0x87, 0xbf, 0xe9},
		Color{0x4c, 0xc9, 0xf0},
		Color{0x43, 0x61, 0xee},
	),
	"newsprint": newPalette("newsprint", false,
		Color{0xf4, 0xf1, 0xea},
		Color{0xb8, 0xb6, 0xb1},
		Color{0x9a, 0x98, 0x94},
		Color{0x78, 0x77, 0x74},
		Color{0x56, 0x55, 0x53},
		Color{0x33, 0x33, 0x32},
		Color{0x8a, 0x1c, 0x1c},
		Color{0x2b, 0x4c, 0x7e},
	),
	"terminal": newPalette("terminal", true,
		Color{0x02, 0x08, 0x03},
		Color{0x26, 0x4d, 0x2f},
		Color{0x38, 0x70, 0x45},
		Color{0x4c, 0x97, 0x5e},
		Color{0x61, 0xbf, 0x77},
		Color{0x75, 0xe6, 0x90},
		Color{0x39, 0xff, 0x14},
		Color{0xff, 0x50, 0x50},
	),
}

func newPalette(name string, dark bool, colors ...Color) Palette {
	p := Palette{
		Name:       name,
		Dark:       dark,
		Background: colors[0],
		Colors:     colors,
	}
	p.labs = make([]lab, len(colors))
	for i, c := range colors {
		p.labs[i] = toOKLab(c)
	}
	// The tonal ramp is the background plus the shade stops. Accents sit
	// outside it so they can be used for small highlights without tinting
	// every midtone.
	rampSize := len(colors)
	if rampSize > 6 {
		rampSize = 6
	}
	p.ramp = make([]Color, rampSize)
	copy(p.ramp, colors[:rampSize])
	sort.SliceStable(p.ramp, func(i, j int) bool {
		return toOKLab(p.ramp[i]).l < toOKLab(p.ramp[j]).l
	})
	p.rampLabs = make([]lab, rampSize)
	for i, c := range p.ramp {
		p.rampLabs[i] = toOKLab(c)
	}
	return p
}

// Names returns the built-in palette names in display order.
func Names() []string {
	return append([]string(nil), order...)
}

// Get returns a built-in palette by name.
func Get(name string) (Palette, bool) {
	p, ok := named[name]
	return p, ok
}

// Exists reports whether name is a built-in palette.
func Exists(name string) bool {
	_, ok := named[name]
	return ok
}

// Nearest returns the closest palette colour to c in OKLab.
func (p Palette) Nearest(c Color) Color {
	return p.nearestLab(toOKLab(c))
}

// NearestDither returns a palette colour after applying an ordered Bayer
// threshold. It breaks gradients into a stable pattern instead of flat bands.
func (p Palette) NearestDither(c Color, x, y int) Color {
	l := toOKLab(c)
	const amount = 0.10
	t := (float64(bayer4[y&3][x&3]) + 0.5) / 16
	l.l += (t - 0.5) * amount
	l.a += (t - 0.5) * amount * 0.3
	l.b += (t - 0.5) * amount * 0.3
	return p.nearestLab(l)
}

// Tone maps luminance in [0,1] onto the palette's background-to-shade ramp.
// Unlike Nearest it never jumps to an accent colour, so photographs keep a
// coherent look instead of picking up stray orange or cyan midtones.
func (p Palette) Tone(l float64) Color {
	if len(p.ramp) == 0 {
		return p.Background
	}
	if len(p.ramp) == 1 {
		return p.ramp[0]
	}
	pos := clamp01(l) * float64(len(p.ramp)-1)
	i := int(math.Floor(pos))
	if i >= len(p.ramp)-1 {
		return p.ramp[len(p.ramp)-1]
	}
	frac := pos - float64(i)
	return fromOKLab(mixLab(p.rampLabs[i], p.rampLabs[i+1], frac))
}

// ToneDither is Tone with an ordered Bayer threshold between the two nearest
// ramp stops. It is stable and avoids the speckle of dithering against a whole
// palette.
func (p Palette) ToneDither(l float64, x, y int) Color {
	if len(p.ramp) == 0 {
		return p.Background
	}
	if len(p.ramp) == 1 {
		return p.ramp[0]
	}
	pos := clamp01(l) * float64(len(p.ramp)-1)
	i := int(math.Floor(pos))
	if i >= len(p.ramp)-1 {
		return p.ramp[len(p.ramp)-1]
	}
	frac := pos - float64(i)
	threshold := (float64(bayer4[y&3][x&3]) + 0.5) / 16
	if frac > threshold {
		i++
	}
	return p.ramp[i]
}

// Ramp returns the palette's tonal stops from darkest to lightest.
func (p Palette) Ramp() []Color {
	return append([]Color(nil), p.ramp...)
}

func mixLab(a, b lab, t float64) lab {
	return lab{
		l: a.l + (b.l-a.l)*t,
		a: a.a + (b.a-a.a)*t,
		b: a.b + (b.b-a.b)*t,
	}
}

func (p Palette) nearestLab(c lab) Color {
	if len(p.Colors) == 0 {
		return Color{}
	}
	labs := p.labs
	if len(labs) != len(p.Colors) {
		labs = make([]lab, len(p.Colors))
		for i, pc := range p.Colors {
			labs[i] = toOKLab(pc)
		}
	}
	best := 0
	bestDist := math.Inf(1)
	for i, pc := range labs {
		dl := c.l - pc.l
		da := c.a - pc.a
		db := c.b - pc.b
		dist := dl*dl + da*da + db*db
		if dist < bestDist {
			bestDist = dist
			best = i
		}
	}
	return p.Colors[best]
}

var bayer4 = [4][4]int{
	{0, 8, 2, 10},
	{12, 4, 14, 6},
	{3, 11, 1, 9},
	{15, 7, 13, 5},
}

func srgbToLinear(v float64) float64 {
	if v <= 0.04045 {
		return v / 12.92
	}
	return math.Pow((v+0.055)/1.055, 2.4)
}

func linearToSRGB(v float64) float64 {
	if v <= 0.0031308 {
		return 12.92 * v
	}
	return 1.055*math.Pow(v, 1/2.4) - 0.055
}

func toOKLab(c Color) lab {
	r := srgbToLinear(float64(c.R) / 255)
	g := srgbToLinear(float64(c.G) / 255)
	b := srgbToLinear(float64(c.B) / 255)

	l := 0.4122214708*r + 0.5363325363*g + 0.0514459929*b
	m := 0.2119034982*r + 0.6806995451*g + 0.1073969566*b
	s := 0.0883024619*r + 0.2817188376*g + 0.6299787005*b

	l = math.Cbrt(l)
	m = math.Cbrt(m)
	s = math.Cbrt(s)

	return lab{
		l: 0.2104542553*l + 0.7936177850*m - 0.0040720468*s,
		a: 1.9779984951*l - 2.4285922050*m + 0.4505937099*s,
		b: 0.0259040371*l + 0.7827717662*m - 0.8086757660*s,
	}
}

func fromOKLab(c lab) Color {
	l := c.l + 0.3963377774*c.a + 0.2158037573*c.b
	m := c.l - 0.1055613458*c.a - 0.0638541728*c.b
	s := c.l - 0.0894841775*c.a - 1.2914855480*c.b

	l = l * l * l
	m = m * m * m
	s = s * s * s

	r := +4.0767416621*l - 3.3077115913*m + 0.2309699292*s
	g := -1.2684380046*l + 2.6097574011*m - 0.3413193965*s
	b := -0.0041960863*l - 0.7034186147*m + 1.7076147010*s

	return Color{
		R: toByte(linearToSRGB(clamp01(r))),
		G: toByte(linearToSRGB(clamp01(g))),
		B: toByte(linearToSRGB(clamp01(b))),
	}
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func toByte(v float64) uint8 {
	return uint8(math.Round(clamp01(v) * 255))
}
