package render

import (
	"bufio"
	"fmt"
	"html"
	"io"
	"strings"
	"sync"
)

// WriteText writes plain characters, trimming trailing spaces on each row.
// Paint output falls back to its tone-mapped glyphs because plain text cannot
// draw cell backgrounds.
func WriteText(w io.Writer, art *Art) error {
	chars := art.Chars
	if len(art.TextChars) == len(art.Chars) {
		chars = art.TextChars
	}
	bw := bufio.NewWriter(w)
	for y := 0; y < art.Rows; y++ {
		line := make([]rune, art.Cols)
		for x := 0; x < art.Cols; x++ {
			line[x] = chars[y*art.Cols+x]
		}
		if _, err := bw.WriteString(strings.TrimRight(string(line), " ")); err != nil {
			return err
		}
		if err := bw.WriteByte('\n'); err != nil {
			return err
		}
	}
	return bw.Flush()
}

// WriteTerminal writes ANSI art. colour is one of none, ansi256 or truecolor.
//
// Escape sequences are only emitted when the colour changes, and a single
// reset closes each coloured row. The previous version reset after every cell,
// which made coloured output several times larger than it needed to be.
func WriteTerminal(w io.Writer, art *Art, colour string) error {
	bw := bufio.NewWriter(w)

	if colour == "none" {
		chars := art.Chars
		if len(art.TextChars) == len(art.Chars) {
			chars = art.TextChars
		}
		for y := 0; y < art.Rows; y++ {
			for x := 0; x < art.Cols; x++ {
				if _, err := bw.WriteRune(chars[y*art.Cols+x]); err != nil {
					return err
				}
			}
			if err := bw.WriteByte('\n'); err != nil {
				return err
			}
		}
		return bw.Flush()
	}

	var lastFG, lastBG Color
	var fgSet, bgSet bool
	for y := 0; y < art.Rows; y++ {
		coloured := false
		for x := 0; x < art.Cols; x++ {
			i := y*art.Cols + x
			ch := art.Chars[i]
			if art.Halfblock {
				if !fgSet || art.FG[i] != lastFG {
					if _, err := fmt.Fprintf(bw, "\x1b[38;%sm", sgr(art.FG[i], colour)); err != nil {
						return err
					}
					lastFG, fgSet = art.FG[i], true
				}
				if !bgSet || art.BG[i] != lastBG {
					if _, err := fmt.Fprintf(bw, "\x1b[48;%sm", sgr(art.BG[i], colour)); err != nil {
						return err
					}
					lastBG, bgSet = art.BG[i], true
				}
				coloured = true
			} else if art.FillMask != nil {
				if art.FillMask[i] {
					if !bgSet || art.BG[i] != lastBG {
						if _, err := fmt.Fprintf(bw, "\x1b[48;%sm", sgr(art.BG[i], colour)); err != nil {
							return err
						}
						lastBG, bgSet = art.BG[i], true
					}
					coloured = true
				}
				if ch != ' ' {
					if !fgSet || art.FG[i] != lastFG {
						if _, err := fmt.Fprintf(bw, "\x1b[38;%sm", sgr(art.FG[i], colour)); err != nil {
							return err
						}
						lastFG, fgSet = art.FG[i], true
					}
					coloured = true
				}
			} else if ch != ' ' {
				if !fgSet || art.FG[i] != lastFG {
					if _, err := fmt.Fprintf(bw, "\x1b[38;%sm", sgr(art.FG[i], colour)); err != nil {
						return err
					}
					lastFG, fgSet = art.FG[i], true
				}
				coloured = true
			}
			if _, err := bw.WriteRune(ch); err != nil {
				return err
			}
		}
		if coloured {
			if _, err := bw.WriteString("\x1b[0m"); err != nil {
				return err
			}
			fgSet, bgSet = false, false
		}
		if err := bw.WriteByte('\n'); err != nil {
			return err
		}
	}
	return bw.Flush()
}

// WriteHTML writes a colour-preserving fragment: one span per coloured glyph,
// suitable for embedding inside a <pre> element. Monochrome art is rendered as
// escaped plain text.
func WriteHTML(w io.Writer, art *Art) error {
	bw := bufio.NewWriter(w)
	for y := 0; y < art.Rows; y++ {
		for x := 0; x < art.Cols; x++ {
			i := y*art.Cols + x
			ch := art.Chars[i]
			switch {
			case art.Halfblock:
				fg := hex(art.FG[i])
				bg := hex(art.BG[i])
				if _, err := fmt.Fprintf(bw, `<span style="color:%s;background:%s">%s</span>`, fg, bg, html.EscapeString(string(ch))); err != nil {
					return err
				}
			case art.FillMask != nil:
				switch {
				case art.FillMask[i] && ch == ' ':
					if _, err := fmt.Fprintf(bw, `<span style="background:%s"> </span>`, hex(art.BG[i])); err != nil {
						return err
					}
				case art.FillMask[i]:
					if _, err := fmt.Fprintf(bw, `<span style="color:%s;background:%s">%s</span>`, hex(art.FG[i]), hex(art.BG[i]), html.EscapeString(string(ch))); err != nil {
						return err
					}
				case ch == ' ':
					if err := bw.WriteByte(' '); err != nil {
						return err
					}
				default:
					if _, err := fmt.Fprintf(bw, `<span style="color:%s">%s</span>`, hex(art.FG[i]), html.EscapeString(string(ch))); err != nil {
						return err
					}
				}
			case ch == ' ':
				if err := bw.WriteByte(' '); err != nil {
					return err
				}
			case art.Monochrome:
				if _, err := bw.WriteString(html.EscapeString(string(ch))); err != nil {
					return err
				}
			default:
				if _, err := fmt.Fprintf(bw, `<span style="color:%s">%s</span>`, hex(art.FG[i]), html.EscapeString(string(ch))); err != nil {
					return err
				}
			}
		}
		if err := bw.WriteByte('\n'); err != nil {
			return err
		}
	}
	return bw.Flush()
}

// WriteHTMLDocument writes a complete standalone HTML page with the fragment
// inside a <pre>. Paint output uses its own base colour so transparent and
// light-background images do not sit on a mismatched page.
func WriteHTMLDocument(w io.Writer, art *Art, title string) error {
	background, foreground := "#070809", "#ece8df"
	if art.FillMask != nil {
		background = hex(art.Base)
		if isLight(art.Base) {
			foreground = "#101317"
		}
	}
	bw := bufio.NewWriter(w)
	if _, err := fmt.Fprintf(bw, "<!doctype html>\n<meta charset=\"utf-8\">\n<title>%s</title>\n", html.EscapeString(title)); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(bw, "<style>body{margin:0;background:%s;color:%s}pre{margin:0;padding:24px;font-family:ui-monospace,SFMono-Regular,Menlo,Consolas,monospace;font-size:12px;line-height:1.15;white-space:pre}</style>\n<pre>", background, foreground); err != nil {
		return err
	}
	if err := WriteHTML(bw, art); err != nil {
		return err
	}
	if _, err := bw.WriteString("</pre>\n"); err != nil {
		return err
	}
	return bw.Flush()
}

func hex(c Color) string {
	return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B)
}

func isLight(c Color) bool {
	return (int(c.R)*299+int(c.G)*587+int(c.B)*114)/1000 > 140
}

func sgr(c Color, mode string) string {
	if mode == "ansi256" {
		return "5;" + fmt.Sprint(ansi256(c))
	}
	return fmt.Sprintf("2;%d;%d;%d", c.R, c.G, c.B)
}

// ansi256Palette holds the 6x6x6 colour cube (indices 16-231) followed by the
// 24-step grey ramp (232-255). The first 16 entries are terminal-dependent and
// are deliberately skipped.
var (
	ansi256Palette [240]Color
	ansi256Once    sync.Once
)

func initAnsi256() {
	levels := [6]uint8{0, 95, 135, 175, 215, 255}
	i := 0
	for _, r := range levels {
		for _, g := range levels {
			for _, b := range levels {
				ansi256Palette[i] = Color{r, g, b}
				i++
			}
		}
	}
	for step := 0; step < 24; step++ {
		v := uint8(8 + step*10)
		ansi256Palette[i] = Color{v, v, v}
		i++
	}
}

// ansi256 returns the nearest entry in the 240-colour palette.
func ansi256(c Color) int {
	ansi256Once.Do(initAnsi256)
	best := 16
	bestDist := 1 << 30
	for i, p := range ansi256Palette {
		dr := int(c.R) - int(p.R)
		dg := int(c.G) - int(p.G)
		db := int(c.B) - int(p.B)
		dist := dr*dr + dg*dg + db*db
		if dist < bestDist {
			bestDist = dist
			best = 16 + i
		}
	}
	return best
}
