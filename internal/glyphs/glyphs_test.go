package glyphs

import "testing"

func TestRampIsMonotonicInCoverage(t *testing.T) {
	set, err := Load(DefaultFont(), " .|/#", 8, 16)
	if err != nil {
		t.Fatal(err)
	}
	ramp := set.Ramp()
	if len(ramp) < 3 {
		t.Fatalf("got %d glyphs", len(ramp))
	}
	for i := 1; i < len(ramp); i++ {
		if ramp[i].Coverage < ramp[i-1].Coverage {
			t.Fatalf("ramp not monotonic at %d: %v < %v", i, ramp[i].Coverage, ramp[i-1].Coverage)
		}
	}
	if ramp[0].Rune != ' ' || ramp[0].Coverage != 0 {
		t.Fatalf("first glyph = %q coverage %v", ramp[0].Rune, ramp[0].Coverage)
	}
}

func TestBestMatchesVerticalBar(t *testing.T) {
	set, err := Load(DefaultFont(), " |", 8, 16)
	if err != nil {
		t.Fatal(err)
	}
	var bar *Glyph
	for i := range set.Glyphs {
		if set.Glyphs[i].Rune == '|' {
			bar = &set.Glyphs[i]
		}
	}
	if bar == nil {
		t.Fatal("vertical bar missing")
	}
	got := set.Best(bar.Bitmap, false)
	if got == nil || got.Rune != '|' {
		t.Fatalf("Best = %v, want |", got)
	}
}

func BenchmarkBest(b *testing.B) {
	set, err := Load(DefaultFont(), DefaultCharset, 8, 16)
	if err != nil {
		b.Fatal(err)
	}
	tile := make([]float32, 8*16)
	for i := range tile {
		tile[i] = float32(i%8) / 8
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = set.Best(tile, false)
	}
	b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N), "ns/cell")
}
