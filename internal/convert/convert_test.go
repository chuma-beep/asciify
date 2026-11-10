package convert

import (
	"os"
	"path/filepath"
	"testing"

	"asciify/internal/glyphs"
)

func TestConvertGoldenRamp(t *testing.T) {
	image, err := os.ReadFile(filepath.Join("..", "..", "testdata", "golden.png"))
	if err != nil {
		t.Fatal(err)
	}

	opts := DefaultOptions()
	opts.Cols = 16
	opts.Mode = "ramp"
	art, err := Convert(image, opts)
	if err != nil {
		t.Fatal(err)
	}

	want := "     @@@@@@00000\n     @@@@@@00000\n     @@@@@@00000\n     @@@@@@00000\n"
	if got := Text(art); got != want {
		t.Fatalf("golden mismatch:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestConvertRejectsInvalidOptions(t *testing.T) {
	opts := DefaultOptions()
	opts.Cols = 2
	if _, err := Convert([]byte{}, opts); err == nil {
		t.Fatal("expected an error for cols below 4")
	}

	opts = DefaultOptions()
	opts.Mode = "painted"
	if _, err := Convert([]byte{}, opts); err == nil {
		t.Fatal("expected an error for an unknown mode")
	}
}

func TestPNGOutput(t *testing.T) {
	image, err := os.ReadFile(filepath.Join("..", "..", "testdata", "circle.png"))
	if err != nil {
		t.Fatal(err)
	}

	opts := DefaultOptions()
	opts.Cols = 24
	art, err := Convert(image, opts)
	if err != nil {
		t.Fatal(err)
	}
	png, err := PNG(art, glyphs.DefaultFont())
	if err != nil {
		t.Fatal(err)
	}
	if len(png) == 0 {
		t.Fatal("PNG output is empty")
	}
}
