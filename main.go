package main

import (
	"image"
	"image/color"
	"image/jpeg"
	"log"
	"os"
	"unicode/utf8"

	"gioui.org/app"
	"gioui.org/f32"
	"gioui.org/font/opentype"
	"gioui.org/io/pointer"
	"gioui.org/io/profile"
	"gioui.org/io/system"
	"gioui.org/op"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	
	"golang.org/x/image/math/fixed"
	"golang.org/x/image/font/gofont/goregular"
)

func main() {
	go func() {
		w := app.NewWindow(app.Title("hello gio"), app.Size(unit.Px(640), unit.Px(480)))
		if err := loop(w); err != nil {
			log.Fatal(err)
		}
	}()
	app.Main()
}

func drawImage(ops *op.Ops, imgop paint.ImageOp, x, y, w, h int) {
	// Use this for ImageCmd
	var stack op.StackOp
	stack.Push(ops)

	imgop.Add(ops)

	var pnt paint.PaintOp
	pnt.Rect.Min = f32.Point{float32(x), float32(y)}
	pnt.Rect.Max = f32.Point{float32(x + w), float32(y + h)}

	pnt.Add(ops)
	stack.Pop()
}

func textPadding(lines []text.Line) (padding image.Rectangle) {
	if len(lines) == 0 {
		return
	}
	first := lines[0]
	if d := first.Ascent + first.Bounds.Min.Y; d < 0 {
		padding.Min.Y = d.Ceil()
	}
	last := lines[len(lines)-1]
	if d := last.Bounds.Max.Y - last.Descent; d > 0 {
		padding.Max.Y = d.Ceil()
	}
	if d := first.Bounds.Min.X; d < 0 {
		padding.Min.X = d.Ceil()
	}
	if d := first.Bounds.Max.X - first.Width; d > 0 {
		padding.Max.X = d.Ceil()
	}
	return
}

func clipLine(line text.Line, clip image.Rectangle) (text.String, f32.Point, bool) {
	off := fixed.Point26_6{X: fixed.I(0), Y: fixed.I(line.Ascent.Ceil())}
	str := line.Text
	for len(str.Advances) > 0 {
		adv := str.Advances[0]
		if (off.X + adv + line.Bounds.Max.X - line.Width).Ceil() >= clip.Min.X {
			break
		}
		off.X += adv
		_, s := utf8.DecodeRuneInString(str.String)
		str.String = str.String[s:]
		str.Advances = str.Advances[1:]
	}
	n := 0
	endx := off.X
	for i, adv := range str.Advances {
		if (endx + line.Bounds.Min.X).Floor() > clip.Max.X {
			str.String = str.String[:n]
			str.Advances = str.Advances[:i]
			break
		}
		_, s := utf8.DecodeRuneInString(str.String[n:])
		n += s
		endx += adv
	}
	offf := f32.Point{X: float32(off.X) / 64, Y: float32(off.Y) / 64}
	return str, offf, true
}

type styleFace struct {
	shaper *text.Shaper
	size   int
}

func styleNewFace(size int) (*styleFace, error) {
	fnt, err := opentype.Parse(goregular.TTF)
	must(err)
	shaper := &text.Shaper{}
	shaper.Register(text.Font{}, fnt)
	return &styleFace{shaper, size}, nil
}

func (face *styleFace) Px(v unit.Value) int {
	return face.size
}

func drawText(ops *op.Ops, face *styleFace, x, y, w, h int, str string) {
	// Use this for TextCmd

	txt := face.shaper.Layout(face, text.Font{}, str, text.LayoutOptions{MaxWidth: 1e6, SingleLine: true})
	if len(txt.Lines) <= 0 {
		return
	}

	bounds := image.Point{X: txt.Lines[0].Width.Ceil(), Y: (txt.Lines[0].Ascent + txt.Lines[0].Descent).Ceil()}
	if bounds.X > w {
		bounds.X = w
	}
	if bounds.Y > h {
		bounds.Y = h
	}
	clip := textPadding(txt.Lines)
	clip.Max = clip.Max.Add(bounds)

	txtstr, off, ok := clipLine(txt.Lines[0], clip)
	if !ok {
		return
	}

	var stack op.StackOp
	stack.Push(ops)
	paint.ColorOp{color.RGBA{0x00, 0x00, 0x00, 0xff}}.Add(ops)

	op.TransformOp{}.Offset(off).Add(ops)
	face.shaper.Shape(face, text.Font{}, txtstr).Add(ops)

	lclip := toRectF(clip).Sub(off)

	var pnt paint.PaintOp
	pnt.Rect = lclip

	pnt.Add(ops)

	stack.Pop()
}

func toRectF(r image.Rectangle) f32.Rectangle {
	return f32.Rectangle{Min: f32.Point{X: float32(r.Min.X), Y: float32(r.Min.Y)}, Max: f32.Point{X: float32(r.Max.X), Y: float32(r.Max.Y)}}
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func loop(w *app.Window) error {
	fh, err := os.Open("../../nucular/_examples/demo/rob.pike.mixtape.jpg")
	must(err)
	defer fh.Close()

	img, err := jpeg.Decode(fh)
	must(err)

	imgop := paint.NewImageOp(img)

	face, err := styleNewFace(16)
	must(err)

	pos := f32.Point{150, 150}
	var ops op.Ops
	for {
		e := <-w.Events()
		switch e := e.(type) {
		case system.DestroyEvent:
			return e.Err
		case system.FrameEvent:
			ops.Reset()

			profileStr := ""
			profile.Op{"blah"}.Add(&ops)
			q := w.Queue()
			for _, e := range q.Events("blah") {
				if e, ok := e.(profile.Event); ok {
					profileStr = e.Timings
				}
			}
	
			drawImage(&ops, imgop, int(pos.X), int(pos.Y), img.Bounds().Dx(), img.Bounds().Dy())

			drawText(&ops, face, 10, 10, e.Size.X, 100, profileStr)
			
			e.Frame(&ops)
		case pointer.Event:
			pos = e.Position
		}
	}
}
