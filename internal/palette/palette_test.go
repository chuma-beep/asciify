package palette

import "testing"

func diff(a, b uint8) int {
	if a > b {
		return int(a - b)
	}
	return int(b - a)
}

func TestOKLabRoundTrip(t *testing.T) {
	cases := []Color{
		{0, 0, 0},
		{255, 255, 255},
		{255, 0, 0},
		{51, 200, 204},
		{120, 90, 40},
	}
	for _, c := range cases {
		got := fromOKLab(toOKLab(c))
		if diff(got.R, c.R) > 2 || diff(got.G, c.G) > 2 || diff(got.B, c.B) > 2 {
			t.Fatalf("round trip %v -> %v", c, got)
		}
	}
}

func TestNearestFindsDarkAndLight(t *testing.T) {
	p, ok := Get("mono")
	if !ok {
		t.Fatal("mono palette missing")
	}
	dark := p.Nearest(Color{18, 18, 18})
	light := p.Nearest(Color{244, 244, 244})
	if dark.R > 40 {
		t.Fatalf("dark grey matched %v, want a near-black stop", dark)
	}
	if light.R < 200 {
		t.Fatalf("light grey matched %v, want a near-white stop", light)
	}
}

func TestDitherIsDeterministicAndVaries(t *testing.T) {
	p, _ := Get("abyss")
	input := Color{120, 120, 120}
	seen := map[Color]bool{}
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			first := p.NearestDither(input, x, y)
			second := p.NearestDither(input, x, y)
			if first != second {
				t.Fatalf("dither is not deterministic at %d,%d: %v vs %v", x, y, first, second)
			}
			seen[first] = true
		}
	}
	if len(seen) < 2 {
		t.Fatalf("ordered dither produced only one colour for a mid grey: %v", seen)
	}
}

func TestNamesAndLookup(t *testing.T) {
	names := Names()
	if len(names) != 8 {
		t.Fatalf("Names() = %v", names)
	}
	for _, name := range names {
		p, ok := Get(name)
		if !ok || p.Name != name || len(p.Colors) == 0 {
			t.Fatalf("bad palette %q: %+v", name, p)
		}
		if p.Colors[0] != p.Background {
			t.Fatalf("%q background does not match first colour", name)
		}
	}
	if Exists("missing") {
		t.Fatal("Exists accepted an unknown palette")
	}
}
