package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGoldenRamp(t *testing.T) {
	out := filepath.Join(t.TempDir(), "golden.txt")
	if err := run([]string{"--cols", "16", "--mode", "ramp", "--out", out, "../../testdata/golden.png"}); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	want := "     @@@@@@00000\n     @@@@@@00000\n     @@@@@@00000\n     @@@@@@00000\n"
	if string(got) != want {
		t.Fatalf("golden mismatch:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestLineArtShapeKeepsEdges(t *testing.T) {
	out := filepath.Join(t.TempDir(), "lineart.txt")
	if err := run([]string{"--cols", "60", "--mode", "shape", "--invert", "--out", out, "../../testdata/lineart.png"}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	found := 0
	for _, edge := range []string{"/", "\\", "|", "_", "-"} {
		if strings.Contains(text, edge) {
			found++
		}
	}
	if found < 3 {
		t.Fatalf("line-art output did not keep enough edge glyphs:\n%s", text)
	}
}

func TestCircleKeepsSquareAspect(t *testing.T) {
	out := filepath.Join(t.TempDir(), "circle.txt")
	if err := run([]string{"--cols", "40", "--mode", "ramp", "--out", out, "../../testdata/circle.png"}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
	if len(lines) < 19 || len(lines) > 21 {
		t.Fatalf("circle rendered %d rows for 40 columns, want about 20", len(lines))
	}
	maxWidth := 0
	for _, line := range lines {
		if n := len([]rune(line)); n > maxWidth {
			maxWidth = n
		}
	}
	if maxWidth > 40 {
		t.Fatalf("circle row width %d exceeds 40 columns", maxWidth)
	}
}
