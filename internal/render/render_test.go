package render

import (
	"bytes"
	"image/png"
	"strings"
	"testing"

	"asciify/internal/glyphs"
	"asciify/internal/pipeline"
)

func testSet(t *testing.T, charset string) *glyphs.Set {
	t.Helper()
	set, err := glyphs.Load(glyphs.DefaultFont(), charset, 8, 16)
	if err != nil {
		t.Fatal(err)
	}
	return set
}

func TestRampUsesMonotonicCoverage(t *testing.T) {
	set := testSet(t, " .:-=+*#%@")
	values := []float32{0, 0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 1}
	grid := &pipeline.Grid{Cols: 10, Rows: 1, L: values,
		R: make([]float32, len(values)), G: make([]float32, len(values)), B: make([]float32, len(values))}
	art := Ramp(grid, set, false, false)
	coverages := make(map[rune]float32)
	for _, g := range set.Glyphs {
		coverages[g.Rune] = g.Coverage
	}
	last := float32(-1)
	for _, ch := range art.Chars {
		c := coverages[ch]
		if c < last {
			t.Fatalf("ramp went backwards at %q: %f < %f", ch, c, last)
		}
		last = c
	}
}

func TestShapeMatchesVerticalBar(t *testing.T) {
	set := testSet(t, " |")
	var bar *glyphs.Glyph
	for i := range set.Glyphs {
		if set.Glyphs[i].Rune == '|' {
			bar = &set.Glyphs[i]
		}
	}
	if bar == nil {
		t.Fatal("missing | glyph")
	}
	tiles := &pipeline.Grid{Cols: 8, Rows: 16, L: append([]float32(nil), bar.Bitmap...)}
	grid := &pipeline.Grid{Cols: 1, Rows: 1, L: []float32{bar.Coverage}, R: []float32{0}, G: []float32{0}, B: []float32{0}}
	art := Shape(grid, tiles, set, false)
	if got := art.Chars[0]; got != '|' {
		t.Fatalf("shape chose %q, want |", got)
	}
}

func TestHalfblockCombinesTwoRows(t *testing.T) {
	grid := &pipeline.Grid{Cols: 1, Rows: 2,
		L: []float32{1, 0}, R: []float32{1, 0}, G: []float32{1, 0}, B: []float32{1, 0}}
	art := Halfblock(grid, true)
	if art.Rows != 1 || art.Chars[0] != '▀' {
		t.Fatalf("unexpected halfblock art: rows=%d char=%q", art.Rows, art.Chars[0])
	}
	if art.FG[0] != (Color{255, 255, 255}) || art.BG[0] != (Color{0, 0, 0}) {
		t.Fatalf("unexpected halfblock colours: fg=%v bg=%v", art.FG[0], art.BG[0])
	}
}

func TestWriteTextTrimsTrailingSpaces(t *testing.T) {
	art := &Art{Cols: 4, Rows: 2, Chars: []rune("a    bc ")}
	var out bytes.Buffer
	if err := WriteText(&out, art); err != nil {
		t.Fatal(err)
	}
	if got, want := out.String(), "a\n bc\n"; got != want {
		t.Fatalf("WriteText = %q, want %q", got, want)
	}
}

func TestWritePNG(t *testing.T) {
	art := &Art{Cols: 2, Rows: 1, Chars: []rune{'A', ' '}, FG: []Color{{255, 255, 255}, {}}, Monochrome: true}
	var out bytes.Buffer
	if err := WritePNG(&out, art, glyphs.DefaultFont(), 8, 16); err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(bytes.NewReader(out.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	if b := img.Bounds(); b.Dx() != 16 || b.Dy() != 16 {
		t.Fatalf("PNG size = %dx%d, want 16x16", b.Dx(), b.Dy())
	}
}

func TestTerminalTruecolor(t *testing.T) {
	art := &Art{Cols: 1, Rows: 1, Chars: []rune{'#'}, FG: []Color{{1, 2, 3}}}
	var out bytes.Buffer
	if err := WriteTerminal(&out, art, "truecolor"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "\x1b[38;2;1;2;3m#") {
		t.Fatalf("terminal output missing truecolor sequence: %q", out.String())
	}
}
