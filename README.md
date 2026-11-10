# asciify

[![CI](https://github.com/chuma-beep/asciify/actions/workflows/ci.yml/badge.svg)](https://github.com/chuma-beep/asciify/actions/workflows/ci.yml)
[![Open in GitHub Codespaces](https://github.com/codespaces/badge.svg)](https://codespaces.new/chuma-beep/asciify)

A fast, font-aware image-to-ASCII converter written in Go.

## Browser demo

Try it online at <https://chuma-beep.github.io/asciify/>.

The demo runs entirely in your browser. The image is converted by a WebAssembly
build of the same pipeline, and nothing is uploaded.

Run it locally:

```sh
./web/build.sh
python3 -m http.server 8000 --directory web/dist
# open http://127.0.0.1:8000/
```

The demo supports PNG, JPEG, and GIF input, all CLI flags except `--out`, a
custom monospace font upload, a source preview, one-click Lagoon / Portrait /
Photo / Line art / Blocks presets, live colour preview, paint texture, seed, and
palette controls, an adjustable output font size, and `.txt` / `.html` / `.png`
export.

## Test it online

The repo includes a Go dev container, so you can run it in a browser without
installing anything locally:

1. Open [GitHub Codespaces](https://codespaces.new/chuma-beep/asciify).
2. Wait for the container and `go test ./...` to finish.
3. In the terminal, run the bundled fixtures:

```sh
go run ./cmd/asciify testdata/gradient.png --cols 80 --mode ramp
go run ./cmd/asciify testdata/lineart.png --cols 80 --mode shape --invert
go run ./cmd/asciify testdata/portrait.png --cols 64 --mode shape
go run ./cmd/asciify testdata/photo.png --cols 100 --mode shape --color truecolor
go build -o asciify ./cmd/asciify
./asciify --help
```

GitHub Actions also runs formatting, vet, tests, the race detector, and a release
build on every push.

`asciify` does more than pick a character from a brightness ramp. It converts
the image to linear light, area-averages it into character cells, normalizes the
tonal range, rasterizes the candidate glyphs from a real monospace font, and can
match each cell against those glyphs by shape.

```sh
go build ./cmd/asciify
./asciify photo.jpg
./asciify --cols 120 --mode shape --color truecolor photo.jpg
./asciify --mode ramp --dither --out portrait.txt portrait.png
./asciify --cols 80 --color truecolor --out card.png card.jpg
./asciify --cols 80 --color truecolor --out card.html card.jpg
./asciify --cols 80 --mode paint --color truecolor --out card-paint.png card.jpg
./asciify --cols 120 --mode paint --color truecolor --palette abyss --out lagoon.png portrait.jpg
./asciify --cols 80 --gamma 1.2 --background auto --out logo.html logo.png
```

## Inputs and outputs

Input: PNG, JPEG, or GIF (first frame).

Output:

- terminal by default, sized from `COLUMNS`, then `stty size`, then 120 columns;
- `.txt` when `--out` ends in `.txt` (ANSI codes included when `--color` is set);
- `.html` when `--out` ends in `.html`, with one coloured span per glyph;
- `.png` when `--out` ends in `.png`, rendered with the same font.

## Flags

| Flag | Default | Meaning |
| --- | --- | --- |
| `--cols` | terminal width or 120 | Output columns. Must be at least 4. |
| `--mode` | `shape` | `ramp` for coverage-ramp mapping, `shape` for glyph MSE matching, `paint` for solid cell fills with contrast texture. |
| `--color` | `none` | `none`, `ansi256`, or `truecolor`. |
| `--charset` | built-in set | Candidate characters, in any order. |
| `--font` | embedded Go Mono | Path to a monospace TTF. |
| `--cell-aspect` | `2.0` | Cell height divided by width. |
| `--clip-low` | `0.02` | Lower luminance percentile clipped before normalization. |
| `--clip-high` | `0.98` | Upper luminance percentile clipped before normalization. |
| `--gamma` | `1.0` | Gamma applied after normalization. Values above 1 lift shadows; below 1 deepen them. |
| `--background` | `black` | Composite transparency over `black`, `white`, or `auto` (picks black or white for contrast with the visible subject). |
| `--texture` | `0.6` | Paint ink strength, `0..1`. Lower values keep the fill dominant. |
| `--seed` | `1` | Deterministic paint texture variation. |
| `--palette` | `none` | Curated look for colour output: `none`, `abyss`, `neon`, `mono`, `paper`, `amber`, `blueprint`, `newsprint`, or `terminal`. |
| `--edges` | off | Unsharp-mask sharpening before matching. |
| `--local-contrast` | off | CLAHE-style local contrast normalization. |
| `--dither` | off | Floyd–Steinberg error diffusion in `ramp` mode; ordered dithering between palette tone stops in `paint` mode. |
| `--invert` | off | Invert ink coverage for light-on-dark output. |
| `--halfblock` | off | Use `▀` with foreground/background colors for double vertical resolution. |
| `--out` | terminal | Write `.txt`, `.html`, or `.png` instead of the terminal. |

The default charset is:

```text
.,:;'"`-_~=+*#%@/\|()[]{}<>!1lIiO0
```

## Pipeline

1. **Decode and linearize.** PNG/JPEG/GIF pixels are converted from sRGB to
   linear RGB with a lookup table. Alpha is composited over black, white, or
   automatically over the backdrop that contrasts with the visible subject (see
   `--background`). Luminance is `0.2126R + 0.7152G + 0.0722B` in linear space.
2. **Area-average downsample.** Every output cell averages all source pixels
   that overlap it. No nearest-neighbor. The target rows use
   `--cell-aspect` (default 2.0) so a square image stays circular.
3. **Normalize tone.** Luminance is clipped at the requested percentiles and
   renormalized to `[0,1]`. `--gamma` then applies a tone curve, which is useful
   for lifting a dark portrait or calming a bright photograph. `--local-contrast`
   adds a box-blur based local step, and `--edges` adds a light unsharp mask.
4. **Build glyphs from the font.** The embedded Go Mono TTF (or `--font`) is
   parsed with `golang.org/x/image/font/opentype`. Each candidate glyph is
   rasterized into an 8×16 float bitmap. Ink coverage is measured and the set is
   sorted by coverage, so the ramp is derived from the font rather than
   hardcoded.
5. **Choose glyphs.** `ramp` maps cell luminance to the nearest coverage value,
   with optional Floyd–Steinberg diffusion. `shape` resamples each cell to the
   glyph bitmap size and picks the glyph with the lowest mean-squared error.
   Shape matching also uses a local-contrast normalization and a small tone
   prior; low-variance cells fall back to coverage matching. `paint` fills each
   covered cell with its average colour, then draws a low-contrast glyph chosen
   from a soft texture subset. A blurred local-detail map keeps flat regions
   mostly solid and lets structured regions carry the texture. Partial cells
   draw only a sparse edge glyph. With `--palette`, the fill is mapped onto the
   palette's tonal ramp in OKLab, so the palette supplies the look without
   letting accents tint every midtone.
6. **Render.** Terminal ANSI, plain text, coloured HTML, or a PNG drawn with the
   same font. Paint output uses cell backgrounds in ANSI/HTML/PNG; plain `.txt`
   falls back to tone-mapped glyphs because text cannot draw fills. ANSI output
   only emits a colour sequence when the colour changes, so it stays compact.
   `--halfblock` uses `▀` with foreground/background colours to double vertical
   resolution.

## Paint style

The hand-authored scenes in the parent project treat each cell as a small painted
tile: a solid fill carries colour and silhouette, and a sparse glyph adds surface
texture. `--mode paint` applies that idea to photographs:

- covered cells get a solid fill in their area-averaged colour;
- glyphs are drawn in a contrasting ink at partial strength, so they read as
  material rather than as the only tone channel;
- a blurred detail map leaves flat backgrounds quiet and places more texture on
  edges and faces;
- partial-coverage cells use a single edge glyph instead of a blocky half-fill;
- monochrome `--color none` fills with luminance grey, and plain `.txt` degrades
  to the normal tone ramp.

`--texture` controls ink strength and `--seed` changes the deterministic texture
pattern. The repository includes `scripts/compare.sh`, which renders each fixture
in both `shape` and `paint` and writes `compare/contact-sheet.png` when ImageMagick
is available.

### Curated palettes

`--palette` applies one of the parent project's named looks. Paint fills are
mapped onto the palette's background and first five shade stops, sorted by OKLab
lightness and interpolated in OKLab. The accent colours stay out of the tone
ramp, so they cannot turn skin orange or make a gradient speckle. `--dither`
selects between the two nearest tone stops with a 4×4 Bayer matrix for gentler
banding.

| Palette | Look |
| --- | --- |
| `abyss` | Deep black water with cold white highlights. |
| `neon` | Night-club cyan, magenta, and acid green. |
| `mono` | Neutral greyscale terminal. |
| `paper` | Warm stock with dark ink. |
| `amber` | Phosphor amber on smoked glass. |
| `blueprint` | Cyan drafting lines on navy. |
| `newsprint` | Warm off-white with black ink. |
| `terminal` | Classic green phosphor. |

The same script renders `testdata/photo.png` through every palette and writes
`compare/palette-sheet.png`. The demo's **Lagoon** preset uses `paint`, true
colour, the `abyss` palette, and a texture of `0.65`. The preset leaves
dithering off: the tone ramp already interpolates between stops, so ordered
dithering is only needed when you deliberately want a posterised gradient.

## Shape vs ramp

At 40 columns on the generated `testdata/portrait.png`, shape keeps more of the
face, eyes, nose, and hair-line structure. Ramp is smoother and flatter.

Shape:

```text
              .]]]]]{{[@[%
            {@%%%OO][%O@/@%%
          @%%@@@%0]0%{O%@@@@@%
         %@%@@%@%%@%%%@%@@%%]%+
        %%]@@%@0000000000%0%@]%%
        @%%IO""""""``""""""@%%%%
          %O%              %0O
          ]@ %@@@@    %@@@% @[
          %%  %@@/    %%%@` `%
          %   %"]  %%  ]"%   %
          @O       %%       %%
           %`      O%       {
           `%@@O  @%@% ]@@./`
             "O"`%@@@%@""%@
              `O%@]]]]%@O`
               %   ``   %
               %]]%%%%_]%
           %@@%"        %@%@%
         %@%%O@%#@O@%@0O@%@0{%%@`
        %@ ]@%%%]@%%[]@%%[]@%)O@
        @%#%%%[#%]@[#%%@@@%]@@%%%
```

Ramp:

```text

              .*IO@00@OI*.
            I00000000000000#
          I000000000000000000#
         0000000000000000000000
        :0000000000000000000000;
        ~##00000000``00000000##:
          #0]              *00
          00 ~###]    )###: 00
          0-  0001    (000` `0
          0   *0O  /#  #0(   0
          ;0       (0       #I
           0`      0}       0
           `0000  !0#- 000.0.
             000`0000000000
              `0###00###0.
               0   ``   0
               0III##III0
           :O##0        0##O!
         :0#(0"00/0/00"0(00:00*0`
        }0 00I0000I0000I0000I00I
        0!0(000000000000000000/0)
```

Shape is not always smoother. On a grainy photograph, the `--edges` flag and a
slightly larger `--cell-aspect` often help. Use ramp when you want a softer,
poster-like result.

## Generated samples

The repository includes:

- `testdata/gradient.png` — tonal ramp check;
- `testdata/circle.png` — aspect-ratio check;
- `testdata/lineart.png` — edge/glyph check;
- `testdata/photo.png`, `testdata/portrait.png`, `testdata/portrait-clean.png`,
  `testdata/portrait-photo.png` — portrait fixtures.

Example outputs are in `out/`:

```sh
./asciify --cols 80 --mode ramp --out out/gradient-ramp.txt testdata/gradient.png
./asciify --cols 80 --mode shape --invert --out out/lineart-shape.txt testdata/lineart.png
./asciify --cols 64 --mode shape --out out/portrait-shape.txt testdata/portrait.png
./asciify --cols 64 --mode ramp --out out/portrait-ramp.txt testdata/portrait.png
./asciify --cols 64 --mode shape --color truecolor --out out/portrait-shape.png testdata/portrait.png
```

The line-art sample shows edge glyphs around the circle and strokes. The circle
sample produces roughly square output at the default cell aspect.

## Performance

On a Ryzen 9 6900HX:

- shape-matching inner loop: **3590 ns/cell**, 0 allocations per cell,
  measured with `go test ./internal/glyphs -bench BenchmarkBest -benchmem`;
- 4000×3000 image at 200 columns in shape mode: **0.924 s**.

Downsampling and shape matching are parallelized across output rows.

## Tests

```sh
go test ./...
go test -race ./...
go vet ./...
go test ./internal/glyphs -bench BenchmarkBest -benchmem -run '^$'
```

Coverage includes sRGB conversion, percentile clipping, gamma, transparent
background compositing, alpha coverage resampling, aspect-aware cell rows, area
averaging, monotonic ramp order, vertical-bar glyph matching, a golden-file ramp
test, circle aspect, line-art edge glyphs, paint fills and texture fallbacks,
OKLab palette tone ramps and matching, deterministic ordered dithering,
compact ANSI output, nearest-palette `ansi256`, escaped HTML output,
colour/halfblock/PNG output, and the shape-matching benchmark.

## Layout

```text
cmd/asciify          CLI and flag parsing
cmd/asciify-wasm     GOOS=js GOARCH=wasm entrypoint for the browser demo
internal/convert     shared conversion core used by the CLI and the browser
internal/pipeline    decode, linear light, resampling, contrast, edges
internal/glyphs      embedded font, glyph rasterization, shape matching
internal/render      ramp/shape/paint/halfblock rendering and output formats
web                  browser demo source and build script
testdata             generated fixtures and golden image
out                  checked-in sample output
```

The default font is Go Mono, embedded under
`internal/glyphs/assets/Go-Mono.ttf` with its license beside it. The module
depends only on the standard library and `golang.org/x/image`.

## Known limits

- Shape mode can amplify grain in noisy photos; `--edges` and a cleaner input
  help. Ramp is often smoother, and paint is smoother still because the fill
  carries most of the tone.
- Paint is designed for photographs and illustrations. Line art and transparent
  logos usually read better in shape mode, where glyph edges define the form.
- GIF support reads the first frame only.
- Truecolor terminal support depends on the terminal; `--color none` is the
  safest default.
