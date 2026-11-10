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

	"asciify/internal/convert"
	"asciify/internal/glyphs"
	"asciify/internal/render"
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
	mode := fs.String("mode", "shape", "rendering mode: ramp, shape, or paint")
	colorMode := fs.String("color", "none", "terminal colour: none, ansi256, or truecolor")
	charset := fs.String("charset", glyphs.DefaultCharset, "candidate glyphs")
	fontPath := fs.String("font", "", "path to a monospace TTF (default: embedded Go Mono)")
	cellAspect := fs.Float64("cell-aspect", 2.0, "character cell height divided by width")
	clipLow := fs.Float64("clip-low", 0.02, "lower luminance percentile to clip")
	clipHigh := fs.Float64("clip-high", 0.98, "upper luminance percentile to clip")
	gamma := fs.Float64("gamma", 1.0, "midtone gamma; values above 1 brighten")
	background := fs.String("background", "black", "composite transparency over black, white, or auto")
	texture := fs.Float64("texture", 0.6, "paint glyph ink strength, 0..1")
	seed := fs.Int64("seed", 1, "paint texture variation seed")
	paletteName := fs.String("palette", "none", "curated colours: none, abyss, neon, mono, paper, amber, blueprint, newsprint, terminal")
	edges := fs.Bool("edges", false, "sharpen with an unsharp mask before matching")
	localContrast := fs.Bool("local-contrast", false, "apply local contrast normalization")
	dither := fs.Bool("dither", false, "dither ramp glyphs and paint fills")
	invert := fs.Bool("invert", false, "invert ink coverage for light-on-dark output")
	halfblock := fs.Bool("halfblock", false, "use upper half blocks for double vertical resolution")
	outPath := fs.String("out", "", "write .txt, .html, or .png instead of the terminal")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return errors.New("expected exactly one input image")
	}

	opts := convert.Options{
		Cols:          *cols,
		Mode:          *mode,
		ColorMode:     *colorMode,
		Charset:       *charset,
		CellAspect:    *cellAspect,
		ClipLow:       *clipLow,
		ClipHigh:      *clipHigh,
		Gamma:         *gamma,
		Background:    *background,
		Texture:       *texture,
		Seed:          *seed,
		Palette:       *paletteName,
		Edges:         *edges,
		LocalContrast: *localContrast,
		Dither:        *dither,
		Invert:        *invert,
		Halfblock:     *halfblock,
	}
	if opts.Cols <= 0 {
		opts.Cols = outputColumns(*outPath == "")
	}
	if err := opts.Validate(); err != nil {
		return err
	}

	if *fontPath != "" {
		data, err := os.ReadFile(*fontPath)
		if err != nil {
			return fmt.Errorf("read font: %w", err)
		}
		opts.FontData = data
	}

	input, err := os.ReadFile(fs.Arg(0))
	if err != nil {
		return fmt.Errorf("read input: %w", err)
	}

	if opts.Cols <= 0 {
		opts.Cols = outputColumns(*outPath == "")
	}
	art, err := convert.Convert(input, opts)
	if err != nil {
		return err
	}

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
		return render.WritePNG(out, art, opts.ResolvedFont(), convert.DefaultCellW, convert.DefaultCellH)
	case ".html":
		out, err := os.Create(*outPath)
		if err != nil {
			return fmt.Errorf("create output: %w", err)
		}
		defer out.Close()
		return render.WriteHTMLDocument(out, art, "asciify — "+filepath.Base(fs.Arg(0)))
	default:
		return fmt.Errorf("unsupported output %q: use .txt, .png, or .html", *outPath)
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
	return 120
}
