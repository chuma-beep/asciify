#!/usr/bin/env bash
# Render the same samples with the glyph and paint styles so the two cell
# treatments can be compared side by side.
set -euo pipefail

cd "$(dirname "$0")/.."

out="compare"
mkdir -p "$out"

go build -o "$out/asciify" ./cmd/asciify

samples=(
  testdata/circle.png
  testdata/lineart.png
  testdata/portrait.png
  testdata/photo.png
  testdata/gradient.png
)

pairs=()
for src in "${samples[@]}"; do
  name="$(basename "$src" .png)"
  for mode in shape paint; do
    "$out/asciify" \
      --cols 80 \
      --mode "$mode" \
      --color truecolor \
      --background auto \
      --texture 0.6 \
      --out "$out/$name-$mode.png" \
      "$src"
    pairs+=("$out/$name-$mode.png")
  done
done

if command -v magick >/dev/null 2>&1; then
  magick montage "${pairs[@]}" \
    -tile 2x \
    -geometry +12+12 \
    -background "#070809" \
    "$out/contact-sheet.png"
  echo "wrote $out/contact-sheet.png"
elif command -v montage >/dev/null 2>&1; then
  montage "${pairs[@]}" \
    -tile 2x \
    -geometry +12+12 \
    -background "#070809" \
    "$out/contact-sheet.png"
  echo "wrote $out/contact-sheet.png"
else
  echo "wrote ${#pairs[@]} images in $out (install ImageMagick for a contact sheet)" >&2
fi

# Curated-look sheet: one photographic sample, every palette.
palettes=(none abyss neon mono paper amber blueprint newsprint terminal)
palette_files=()
for name in "${palettes[@]}"; do
  "$out/asciify" \
    --cols 72 \
    --mode paint \
    --color truecolor \
    --palette "$name" \
    --dither \
    --background auto \
    --texture 0.6 \
    --out "$out/palette-$name.png" \
    testdata/photo.png
  palette_files+=("$out/palette-$name.png")
done

if command -v magick >/dev/null 2>&1; then
  magick montage "${palette_files[@]}" \
    -tile 3x \
    -geometry +10+10 \
    -background "#070809" \
    "$out/palette-sheet.png"
  echo "wrote $out/palette-sheet.png"
elif command -v montage >/dev/null 2>&1; then
  montage "${palette_files[@]}" \
    -tile 3x \
    -geometry +10+10 \
    -background "#070809" \
    "$out/palette-sheet.png"
  echo "wrote $out/palette-sheet.png"
fi
