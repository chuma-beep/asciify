// Command asciify converts an image into ASCII art.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"asciify/internal/glyphs"
	"asciify/internal/pipeline"
	"asciify/internal/render"
)

const (
	defaultCellW = 8
	defaultCellH = 16
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "asciify:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	fs := flag.NewFlagSet("asciify", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: asciify [flags] input.(png|jpg|jpeg|gif)")
		fmt.Fprintln(fs.Output())
		fs.PrintDefaults()
	}

	cols := fs.Int("cols", 0, "output columns; 0 uses the terminal width or 120")
	mode := fs.String("mode", "shape", "rendering mode: ramp or shape")
	colorMode := fs.String("color", "none", "terminal colour: none, ansi256, or truecolor")
	charset := fs.String("charset", glyphs.DefaultCharset, "candidate glyphs")
	fontPath := fs.String("font", "", "path to a monospace TTF (default: embedded Go Mono)")
	cellAspect := fs.Float64("cell-aspect", 2.0, "character cell height divided by width")
	clipLow := fs.Float64("clip-low", 0.02, "lower luminance percentile to clip")
	clipHigh := fs.Float64("clip-high", 0.98, "upper luminance percentile to clip")
	edges := fs.Bool("edges", false, "sharpen with an unsharp mask before matching")
	localContrast := fs.Bool("local-contrast", false, "apply local contrast normalization")
	dither := fs.Bool("dither", false, "Floyd-Steinberg dithering in ramp mode")
	invert := fs.Bool("invert", false, "invert ink coverage for light-on-dark output")
	halfblock := fs.Bool("halfblock", false, "use upper half blocks for double vertical resolution")
	outPath := fs.String("out", "", "write .txt or .png instead of the terminal")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return errors.New("expected exactly one input image")
	}
	if *mode != "ramp" && *mode != "shape" {
		return fmt.Errorf("unknown mode %q: use ramp or shape", *mode)
	}
	if *colorMode != "none" && *colorMode != "ansi256" && *colorMode != "truecolor" {
		return fmt.Errorf("unknown colour mode %q: use none, ansi256, or truecolor", *colorMode)
	}
	if *cellAspect <= 0 {
		return errors.New("--cell-aspect must be positive")
	}
	if *clipLow < 0 || *clipHigh > 1 || *clipLow >= *clipHigh {
		return errors.New("--clip-low and --clip-high must satisfy 0 <= low < high <= 1")
	}

	fontData := glyphs.DefaultFont()
	if *fontPath != "" {
		data, err := os.ReadFile(*fontPath)
		if err != nil {
			return fmt.Errorf("read font: %w", err)
		}
		fontData = data
	}

	file, err := os.Open(fs.Arg(0))
	if err != nil {
		return fmt.Errorf("open input: %w", err)
	}
	defer file.Close()
	img, err := pipeline.Decode(file)
	if err != nil {
		return err
	}

	columns := *cols
	if columns <= 0 {
		columns = outputColumns(*outPath == "")
	}
	if columns < 4 {
		return errors.New("--cols must be at least 4")
	}

	rows := pipeline.CellRows(img.Width, img.Height, columns, *cellAspect)
	sampleRows := rows
	if *halfblock {
		sampleRows = rows * 2
	}
	grid := pipeline.Resample(img, columns, sampleRows)

	low, high := pipeline.ClipPercentiles(grid, *clipLow, *clipHigh)
	pipeline.Normalize(grid, low, high)

	var tiles *pipeline.Grid
	var set *glyphs.Set
	if !*halfblock {
		set, err = glyphs.Load(fontData, *charset, defaultCellW, defaultCellH)
		if err != nil {
			return err
		}
		if *mode == "shape" {
			tiles = pipeline.Resample(img, columns*defaultCellW, sampleRows*defaultCellH)
			pipeline.NormalizeValues(tiles.L, low, high)
		}
	}

	if *localContrast {
		pipeline.LocalContrast(grid, 3, 1.8)
		if tiles != nil {
			pipeline.LocalContrast(tiles, 3, 1.8)
		}
	}
	if *edges {
		pipeline.Unsharp(grid, 1.2)
		if tiles != nil {
			pipeline.Unsharp(tiles, 1.2)
		}
	}

	var art *render.Art
	switch {
	case *halfblock:
		art = render.Halfblock(grid, *colorMode != "none")
	case *mode == "shape":
		art = render.Shape(grid, tiles, set, *invert)
	default:
		art = render.Ramp(grid, set, *dither, *invert)
	}
	art.Monochrome = *colorMode == "none"

	if *outPath == "" {
		return render.WriteTerminal(os.Stdout, art, *colorMode)
	}
	switch strings.ToLower(filepath.Ext(*outPath)) {
	case ".txt":
		out, err := os.Create(*outPath)
		if err != nil {
			return fmt.Errorf("create output: %w", err)
		}
		defer out.Close()
		return render.WriteText(out, art)
	case ".png":
		out, err := os.Create(*outPath)
		if err != nil {
			return fmt.Errorf("create output: %w", err)
		}
		defer out.Close()
		return render.WritePNG(out, art, fontData, defaultCellW, defaultCellH)
	default:
		return fmt.Errorf("unsupported output %q: use .txt or .png", *outPath)
	}
}

func outputColumns(terminal bool) int {
	if terminal {
		if value := os.Getenv("COLUMNS"); value != "" {
			if n, err := strconv.Atoi(value); err == nil && n > 0 {
				return n
			}
		}
		if out, err := exec.Command("stty", "size").Output(); err == nil {
			fields := strings.Fields(string(out))
			if len(fields) == 2 {
				if n, err := strconv.Atoi(fields[1]); err == nil && n > 0 {
					return n
				}
			}
		}
	}
	if terminal {
		return 120
	}
	return 120
}
