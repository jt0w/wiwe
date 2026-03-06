package main

import (
	_ "embed"
	"fmt"
	"os"

	gemini "github.com/secretval/wiwe/cmd/wiwe/protocols/gemini"

	"image"
	"image/color"
	"log"
	"strings"

	"gioui.org/app"
	"gioui.org/font"
	_ "gioui.org/font/gofont"
	"gioui.org/font/opentype"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

type State struct {
	Url string
	Res gemini.GeminiResponse
}

var STATE = State{}

//go:embed fonts/Iosevkajt0w-Regular.ttf
var DefaultFontBytes []byte
var DefaultFont = text.FontFace{
	Font: font.Font{
		Typeface: "Iosevka Jt0w",
		Weight:   font.Normal,
		Style:    font.Regular,
	},
	Face: ParseFont(DefaultFontBytes),
}

func ParseFont(b []byte) font.Face {
	face, err := opentype.Parse(b)
	if err != nil {
		panic(err)
	}
	return face
}

func main() {
	prog := os.Args[0]
	args := os.Args[1:]
	if len(args) < 1 {
		fmt.Printf("Usage: <%s> <url>\n", prog)
		fmt.Printf("ERROR: Did not specify url\n")
		os.Exit(1)
	}
	STATE.Url = args[0]

	NewReq()

	go func() {
		w := new(app.Window)
		err := display(w)
		if err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()
	app.Main()
}

var list layout.List

func NewReq() error {
	req, err := gemini.ParseGeminiRequest(STATE.Url, gemini.PORT)
	if err != nil {
		return err
	}

	STATE.Res = gemini.MakeGeminiQuery(req)
	return nil
}

func display(w *app.Window) error {
	theme := material.NewTheme()
	theme.Bg = color.NRGBA{18, 18, 18, 255}
	theme.Shaper = text.NewShaper(text.WithCollection([]font.FontFace{DefaultFont}))
	buf := strings.ToValidUTF8(STATE.Res.Body, "")
	lines := strings.Split(buf, "\n")
	for i, line := range lines {
		lines[i] = cleanLine(line)
	}
	var links []widget.Clickable
	if len(links) != len(lines) {
		links = make([]widget.Clickable, len(lines))
	}
	var ops op.Ops
	lastUrl := STATE.Url
	for {
		ev := w.Event()
		switch e := ev.(type) {
		case app.DestroyEvent:
			return e.Err
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)

			if lastUrl != STATE.Url {
				NewReq()
				buf = strings.ToValidUTF8(STATE.Res.Body, "")
				lines = strings.Split(buf, "\n")
				for i, line := range lines {
					lines[i] = cleanLine(line)
				}
				links = make([]widget.Clickable, len(lines))
				lastUrl = STATE.Url
			}

			DrawRect(clip.Rect{Max: image.Pt(gtx.Constraints.Max.X, gtx.Constraints.Max.Y)}, theme.Bg, &ops)

			list.Axis = layout.Vertical
			list.Layout(gtx, len(lines), func(gtx layout.Context, i int) layout.Dimensions {
				line := lines[i]
				label := material.Label(theme, 16, fmt.Sprintf("%s", line))
				label.Color = color.NRGBA{R: 255, G: 255, B: 255, A: 255}

				if strings.HasPrefix(line, "=>") {
					label.Color = color.NRGBA{R: 125, G: 125, B: 255, A: 255}

					for links[i].Clicked(gtx) {
						url, _ := strings.CutPrefix(line, "=>")
						url = strings.TrimSpace(url)
						url = strings.Fields(url)[0]
						if strings.HasPrefix(url, "gemini://") {
							STATE.Url = url
						} else {
							STATE.Url = fmt.Sprintf("%s/%s", STATE.Url, url)
						}
					}

					return links[i].Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						pointer.CursorPointer.Add(gtx.Ops)
						return label.Layout(gtx)
					})
				}

				return label.Layout(gtx)
			})

			e.Frame(gtx.Ops)
		}
	}

}

func DrawRect(rect clip.Rect, color color.NRGBA, ops *op.Ops) {
	defer rect.Push(ops).Pop()
	paint.ColorOp{Color: color}.Add(ops)
	paint.PaintOp{}.Add(ops)
}

func cleanLine(s string) string {
    return strings.Map(func(r rune) rune {
        if r == '\t' || r == '\r' {
            return ' '
        }
        if r < 32 || r == 127 { // control characters
            return -1 // drop
        }
        return r
    }, s)
}
