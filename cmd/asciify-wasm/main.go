//go:build js && wasm

// Command asciify-wasm exposes the conversion core to the browser.
//
// It is built with GOOS=js GOARCH=wasm and loaded by web/worker.js. The Go
// program installs one function, asciifyConvert(imageBytes, options), which
// returns plain text and, on request, a base64-encoded PNG.
package main

import (
	"encoding/base64"
	"syscall/js"

	"asciify/internal/convert"
)

func main() {
	js.Global().Set("asciifyConvert", js.FuncOf(asciifyConvert))
	js.Global().Set("asciifyReady", true)
	select {}
}

func asciifyConvert(_ js.Value, args []js.Value) any {
	if len(args) == 0 {
		return errorResult("missing image data")
	}

	opts := convert.DefaultOptions()
	if len(args) > 1 && !args[1].IsUndefined() && !args[1].IsNull() {
		opts = readOptions(args[1])
		if fontData := args[1].Get("fontData"); !fontData.IsUndefined() && !fontData.IsNull() {
			opts.FontData = bytesOf(fontData)
		}
	}

	image := bytesOf(args[0])
	art, err := convert.Convert(image, opts)
	if err != nil {
		return errorResult(err.Error())
	}

	result := map[string]any{
		"ok":   true,
		"text": convert.Text(art),
		"cols": art.Cols,
		"rows": art.Rows,
	}
	if opts.ColorMode != "none" || art.FillMask != nil {
		result["ansi"] = convert.Terminal(art, opts.ColorMode)
		result["html"] = convert.HTML(art)
	}
	if boolOption(args, "png") {
		png, err := convert.PNG(art, opts.ResolvedFont())
		if err != nil {
			return errorResult(err.Error())
		}
		result["png"] = base64.StdEncoding.EncodeToString(png)
	}
	return result
}

func readOptions(v js.Value) convert.Options {
	opts := convert.DefaultOptions()
	if v.IsUndefined() || v.IsNull() {
		return opts
	}
	if x := v.Get("cols"); !x.IsUndefined() && !x.IsNull() {
		opts.Cols = x.Int()
	}
	if x := v.Get("mode"); !x.IsUndefined() && !x.IsNull() {
		opts.Mode = x.String()
	}
	if x := v.Get("color"); !x.IsUndefined() && !x.IsNull() {
		opts.ColorMode = x.String()
	}
	if x := v.Get("charset"); !x.IsUndefined() && !x.IsNull() {
		opts.Charset = x.String()
	}
	if x := v.Get("cellAspect"); !x.IsUndefined() && !x.IsNull() {
		opts.CellAspect = x.Float()
	}
	if x := v.Get("clipLow"); !x.IsUndefined() && !x.IsNull() {
		opts.ClipLow = x.Float()
	}
	if x := v.Get("clipHigh"); !x.IsUndefined() && !x.IsNull() {
		opts.ClipHigh = x.Float()
	}
	if x := v.Get("gamma"); !x.IsUndefined() && !x.IsNull() {
		opts.Gamma = x.Float()
	}
	if x := v.Get("background"); !x.IsUndefined() && !x.IsNull() {
		opts.Background = x.String()
	}
	if x := v.Get("texture"); !x.IsUndefined() && !x.IsNull() {
		opts.Texture = x.Float()
	}
	if x := v.Get("seed"); !x.IsUndefined() && !x.IsNull() {
		opts.Seed = int64(x.Int())
	}
	if x := v.Get("palette"); !x.IsUndefined() && !x.IsNull() {
		opts.Palette = x.String()
	}
	if x := v.Get("edges"); !x.IsUndefined() && !x.IsNull() {
		opts.Edges = x.Bool()
	}
	if x := v.Get("localContrast"); !x.IsUndefined() && !x.IsNull() {
		opts.LocalContrast = x.Bool()
	}
	if x := v.Get("dither"); !x.IsUndefined() && !x.IsNull() {
		opts.Dither = x.Bool()
	}
	if x := v.Get("invert"); !x.IsUndefined() && !x.IsNull() {
		opts.Invert = x.Bool()
	}
	if x := v.Get("halfblock"); !x.IsUndefined() && !x.IsNull() {
		opts.Halfblock = x.Bool()
	}
	return opts
}

func boolOption(args []js.Value, name string) bool {
	if len(args) < 2 || args[1].IsUndefined() || args[1].IsNull() {
		return false
	}
	v := args[1].Get(name)
	return !v.IsUndefined() && !v.IsNull() && v.Bool()
}

func bytesOf(v js.Value) []byte {
	// CopyBytesToGo only accepts Uint8Array/Uint8ClampedArray. Callers may
	// hand us an ArrayBuffer (for example through postMessage), so wrap it.
	if v.InstanceOf(js.Global().Get("ArrayBuffer")) {
		v = js.Global().Get("Uint8Array").New(v)
	}
	n := v.Get("byteLength").Int()
	out := make([]byte, n)
	js.CopyBytesToGo(out, v)
	return out
}

func errorResult(message string) map[string]any {
	return map[string]any{"ok": false, "error": message}
}
