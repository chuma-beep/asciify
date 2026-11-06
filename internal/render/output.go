package render

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// WriteText writes plain characters, trimming trailing spaces on each row.
func WriteText(w io.Writer, art *Art) error {
	bw := bufio.NewWriter(w)
	for y := 0; y < art.Rows; y++ {
		line := make([]rune, art.Cols)
		for x := 0; x < art.Cols; x++ {
			line[x] = art.Chars[y*art.Cols+x]
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
func WriteTerminal(w io.Writer, art *Art, colour string) error {
	bw := bufio.NewWriter(w)
	for y := 0; y < art.Rows; y++ {
		for x := 0; x < art.Cols; x++ {
			i := y*art.Cols + x
			ch := art.Chars[i]
			if colour == "none" {
				if _, err := bw.WriteRune(ch); err != nil {
					return err
				}
				continue
			}
			if art.Halfblock {
				if _, err := fmt.Fprintf(bw, "\x1b[38;%sm\x1b[48;%sm", sgr(art.FG[i], colour), sgr(art.BG[i], colour)); err != nil {
					return err
				}
			} else if ch != ' ' {
				if _, err := fmt.Fprintf(bw, "\x1b[38;%sm", sgr(art.FG[i], colour)); err != nil {
					return err
				}
			}
			if _, err := bw.WriteRune(ch); err != nil {
				return err
			}
			if colour != "none" && (art.Halfblock || ch != ' ') {
				if _, err := bw.WriteString("\x1b[0m"); err != nil {
					return err
				}
			}
		}
		if _, err := bw.WriteString("\n"); err != nil {
			return err
		}
	}
	return bw.Flush()
}

func sgr(c Color, mode string) string {
	if mode == "ansi256" {
		return "5;" + fmt.Sprint(ansi256(c))
	}
	return fmt.Sprintf("2;%d;%d;%d", c.R, c.G, c.B)
}

func ansi256(c Color) int {
	r, g, b := int(c.R), int(c.G), int(c.B)
	if abs(r-g) < 10 && abs(g-b) < 10 {
		if r < 8 {
			return 16
		}
		if r > 248 {
			return 231
		}
		return 232 + (r-8)/10
	}
	ri, gi, bi := r*5/255, g*5/255, b*5/255
	return 16 + 36*ri + 6*gi + bi
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
