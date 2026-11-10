#!/usr/bin/env bash
# Build the browser demo into web/dist for local testing or GitHub Pages.
set -euo pipefail

cd "$(dirname "$0")/.."

dist="web/dist"
wasm_exec="$(go env GOROOT)/lib/wasm/wasm_exec.js"
if [ ! -f "$wasm_exec" ]; then
  wasm_exec="$(go env GOROOT)/misc/wasm/wasm_exec.js"
fi
if [ ! -f "$wasm_exec" ]; then
  echo "wasm_exec.js not found in $(go env GOROOT)" >&2
  exit 1
fi

rm -rf "$dist"
mkdir -p "$dist/samples"

cp "$wasm_exec" "$dist/"
GOOS=js GOARCH=wasm go build -trimpath -o "$dist/asciify.wasm" ./cmd/asciify-wasm
cp web/index.html web/styles.css web/app.js web/worker.js "$dist/"
cp \
  testdata/gradient.png \
  testdata/circle.png \
  testdata/lineart.png \
  testdata/portrait.png \
  testdata/photo.png \
  "$dist/samples/"

echo "Built $dist"
