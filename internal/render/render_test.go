package render

import (
	"bytes"
	"image/png"
	"strings"
	"testing"

	"asciify/internal/glyphs"
	"asciify/internal/palette"
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

func TestTerminalSetsColourOncePerRun(t *testing.T) {
	art := &Art{
		Cols: 3, Rows: 1,
		Chars: []rune("###"),
		FG:    []Color{{1, 2, 3}, {1, 2, 3}, {1, 2, 3}},
	}
	var out bytes.Buffer
	if err := WriteTerminal(&out, art, "truecolor"); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if count := strings.Count(got, "\x1b[38;2;1;2;3m"); count != 1 {
		t.Fatalf("colour sequence emitted %d times, want 1: %q", count, got)
	}
	if count := strings.Count(got, "\x1b[0m"); count != 1 {
		t.Fatalf("reset emitted %d times, want 1: %q", count, got)
	}
}

func TestAnsi256NearestPalette(t *testing.T) {
	if got := ansi256(Color{255, 0, 0}); got != 196 {
		t.Fatalf("ansi256 pure red = %d, want 196", got)
	}
	if got := ansi256(Color{128, 128, 128}); got != 244 {
		t.Fatalf("ansi256 mid grey = %d, want 244", got)
	}
}

func TestWriteHTMLPreservesColourAndEscapes(t *testing.T) {
	art := &Art{
		Cols: 3, Rows: 1,
		Chars: []rune{'<', ' ', '#'},
		FG:    []Color{{255, 0, 0}, {}, {1, 2, 3}},
	}
	var out bytes.Buffer
	if err := WriteHTML(&out, art); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if !strings.Contains(got, `<span style="color:#ff0000">&lt;</span>`) {
		t.Fatalf("HTML did not escape or colour the glyph: %q", got)
	}
	if !strings.Contains(got, `<span style="color:#010203">#</span>`) {
		t.Fatalf("HTML missing truecolor span: %q", got)
	}
}

func TestWriteHTMLDocumentMonochrome(t *testing.T) {
	art := &Art{Cols: 2, Rows: 1, Chars: []rune{'<', '>'}, Monochrome: true}
	var out bytes.Buffer
	if err := WriteHTMLDocument(&out, art, "demo"); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if !strings.Contains(got, "<!doctype html>") || !strings.Contains(got, "&lt;&gt;") {
		t.Fatalf("unexpected HTML document: %q", got)
	}
}

func TestPaintFillsAndFeathers(t *testing.T) {
	set := testSet(t, " .:-=+*#%@")
	const cols = 24
	grid := &pipeline.Grid{
		Cols: cols, Rows: 1,
		L: make([]float32, cols),
		R: make([]float32, cols),
		G: make([]float32, cols),
		B: make([]float32, cols),
		A: make([]float32, cols),
	}
	for i := 0; i < cols; i++ {
		grid.L[i] = 0.2 + 0.6*float32(i%5)/4
		grid.R[i], grid.G[i], grid.B[i] = 0.1, 0.2, 0.3
		grid.A[i] = 1
	}
	grid.A[0] = 0.3 // partial edge
	grid.A[1] = 0   // empty

	art := Paint(grid, set, 0.6, 1, false, Color{}, nil, false)
	if len(art.FillMask) != cols {
		t.Fatalf("FillMask length = %d, want %d", len(art.FillMask), cols)
	}
	if art.FillMask[0] || art.FillMask[1] {
		t.Fatal("partial and empty cells must not be filled")
	}
	if !art.FillMask[2] {
		t.Fatal("covered cell was not filled")
	}
	if art.Chars[0] == ' ' || art.Chars[0] == 0 {
		t.Fatal("partial cell did not receive an edge glyph")
	}
	if art.Chars[1] != ' ' {
		t.Fatal("empty cell should stay blank")
	}
	if art.BG[2] == (Color{}) {
		t.Fatal("filled cell has no background colour")
	}
	if art.TextChars[2] == ' ' || art.TextChars[2] == 0 {
		t.Fatal("paint did not build a plain-text fallback")
	}

	textured := false
	for i := 2; i < cols; i++ {
		if art.FillMask[i] && art.Chars[i] != ' ' && art.FG[i] != art.BG[i] {
			textured = true
			break
		}
	}
	if !textured {
		t.Fatal("no filled cell received contrasting texture ink")
	}
}

func TestPaintPaletteToneMapsFills(t *testing.T) {
	set := testSet(t, " .:-=+*#%@")
	pal, ok := palette.Get("abyss")
	if !ok {
		t.Fatal("abyss palette missing")
	}
	ramp := pal.Ramp()

	values := []float32{0, 0.5, 1}
	grid := &pipeline.Grid{
		Cols: len(values), Rows: 1,
		L: values,
		R: []float32{0.2, 0.5, 0.8},
		G: []float32{0.2, 0.5, 0.8},
		B: []float32{0.2, 0.5, 0.8},
		A: []float32{1, 1, 1},
	}
	art := Paint(grid, set, 0.6, 1, false, Color{}, &pal, false)

	want := []palette.Color{ramp[0], pal.Tone(0.5), ramp[len(ramp)-1]}
	for i, c := range want {
		got := art.BG[i]
		if got.R != c.R || got.G != c.G || got.B != c.B {
			t.Fatalf("cell %d fill = %+v, want %+v", i, got, c)
		}
		if !art.FillMask[i] {
			t.Fatalf("cell %d was not filled", i)
		}
	}
}

func TestWriteTextUsesPaintFallback(t *testing.T) {
	art := &Art{
		Cols: 2, Rows: 1,
		Chars:     []rune{'X', 'X'},
		TextChars: []rune{'.', ' '},
	}
	var out bytes.Buffer
	if err := WriteText(&out, art); err != nil {
		t.Fatal(err)
	}
	if got := out.String(); got != ".\n" {
		t.Fatalf("WriteText = %q, want %q", got, ".\n")
	}
}

func TestTerminalPaintDrawsBackground(t *testing.T) {
	art := &Art{
		Cols: 1, Rows: 1,
		Chars:    []rune{'#'},
		FG:       []Color{{255, 255, 255}},
		BG:       []Color{{10, 20, 30}},
		FillMask: []bool{true},
	}
	var out bytes.Buffer
	if err := WriteTerminal(&out, art, "truecolor"); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if !strings.Contains(got, "\x1b[48;2;10;20;30m") || !strings.Contains(got, "\x1b[38;2;255;255;255m") {
		t.Fatalf("paint terminal output missing fill or ink: %q", got)
	}
}

func TestWriteHTMLPaintDrawsBackground(t *testing.T) {
	art := &Art{
		Cols: 1, Rows: 1,
		Chars:    []rune{'#'},
		FG:       []Color{{255, 255, 255}},
		BG:       []Color{{10, 20, 30}},
		FillMask: []bool{true},
	}
	var out bytes.Buffer
	if err := WriteHTML(&out, art); err != nil {
		t.Fatal(err)
	}
	if got := out.String(); !strings.Contains(got, "background:#0a141e") || !strings.Contains(got, "color:#ffffff") {
		t.Fatalf("paint HTML missing fill or ink: %q", got)
	}
}

func TestWritePNGUsesPaintBase(t *testing.T) {
	art := &Art{
		Cols: 1, Rows: 1,
		Chars:    []rune{' '},
		FG:       []Color{{0, 0, 0}},
		BG:       []Color{{0, 0, 0}},
		FillMask: []bool{false},
		Base:     Color{255, 255, 255},
	}
	var out bytes.Buffer
	if err := WritePNG(&out, art, glyphs.DefaultFont(), 4, 8); err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(&out)
	if err != nil {
		t.Fatal(err)
	}
	r, g, b, _ := img.At(0, 0).RGBA()
	if r>>8 != 255 || g>>8 != 255 || b>>8 != 255 {
		t.Fatalf("PNG base = (%d,%d,%d), want white", r>>8, g>>8, b>>8)
	}
}
