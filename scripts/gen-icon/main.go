// Generates a 1024x1024 source icon for MeshManager: a dark rounded tile with
// a server glyph in the accent color. The glyph is the lucide "Server" icon
// (two stacked rounded-rect units, each with an indicator dot) so the app icon
// in the titlebar / taskbar / dock matches the icon shown in the About dialog.
package main

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
)

const S = 1024

func main() {
	img := image.NewRGBA(image.Rect(0, 0, S, S))

	bg := color.RGBA{0x11, 0x15, 0x1f, 0xff}
	tile := color.RGBA{0x16, 0x1b, 0x27, 0xff}
	accent := color.RGBA{0x4c, 0x8d, 0xff, 0xff}

	radius := 180.0
	for y := 0; y < S; y++ {
		for x := 0; x < S; x++ {
			if insideRounded(x, y, 64, S-64, radius) {
				img.Set(x, y, tile)
			} else {
				img.Set(x, y, bg)
			}
		}
	}

	// lucide "Server" geometry lives in a 24x24 viewBox; map it onto the canvas
	// centered, scaled so the 24-unit box stays comfortably inside the tile.
	const sc = 31.0
	lx := func(u float64) float64 { return S/2 + (u-12)*sc }
	ly := func(v float64) float64 { return S/2 + (v-12)*sc }
	stroke := 2.0 * sc // lucide default stroke-width = 2 units

	// Two server units: rect(x2 y2 w20 h8 rx2) and rect(x2 y14 w20 h8 rx2).
	// Center = (12, y+4); half-extents = (10, 4); corner radius = 2.
	strokeRoundRect(img, lx(12), ly(6), 10*sc, 4*sc, 2*sc, stroke/2, accent)
	strokeRoundRect(img, lx(12), ly(18), 10*sc, 4*sc, 2*sc, stroke/2, accent)

	// Indicator dots at (6,6) and (6,18) — drawn as round-capped points.
	drawDisc(img, lx(6), ly(6), stroke/2, accent)
	drawDisc(img, lx(6), ly(18), stroke/2, accent)

	f, err := os.Create(os.Args[1])
	if err != nil {
		panic(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		panic(err)
	}
}

// insideRounded reports whether (x,y) lies inside the square [lo,hi]^2 with
// corners rounded at radius r. The rounded rect is the set of points within r
// of the inner rect [lo+r, hi-r]^2, so we just test distance to that box.
func insideRounded(x, y, lo, hi int, r float64) bool {
	clamp := func(v, mn, mx float64) float64 { return math.Max(mn, math.Min(mx, v)) }
	flo, fhi := float64(lo), float64(hi)
	ix := clamp(float64(x), flo+r, fhi-r)
	iy := clamp(float64(y), flo+r, fhi-r)
	return math.Hypot(float64(x)-ix, float64(y)-iy) <= r
}

// sdRoundRect is the signed distance from a point (relative to the rect center)
// to a rounded rectangle with the given half-extents and corner radius.
// Negative inside, zero on the boundary, positive outside.
func sdRoundRect(px, py, hw, hh, r float64) float64 {
	qx := math.Abs(px) - (hw - r)
	qy := math.Abs(py) - (hh - r)
	return math.Hypot(math.Max(qx, 0), math.Max(qy, 0)) + math.Min(math.Max(qx, qy), 0) - r
}

// strokeRoundRect draws the outline of a rounded rectangle: every pixel within
// halfW of the boundary, with a 1.5px antialiased edge.
func strokeRoundRect(img *image.RGBA, cx, cy, hw, hh, r, halfW float64, c color.RGBA) {
	x0, x1 := int(cx-hw-halfW-2), int(cx+hw+halfW+2)
	y0, y1 := int(cy-hh-halfW-2), int(cy+hh+halfW+2)
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			if x < 0 || y < 0 || x >= S || y >= S {
				continue
			}
			d := math.Abs(sdRoundRect(float64(x)-cx, float64(y)-cy, hw, hh, r))
			if d <= halfW {
				blend(img, x, y, c, 1)
			} else if d <= halfW+1.5 {
				blend(img, x, y, c, (halfW+1.5-d)/1.5)
			}
		}
	}
}

func drawDisc(img *image.RGBA, cx, cy, r float64, c color.RGBA) {
	for y := int(cy - r); y <= int(cy+r); y++ {
		for x := int(cx - r); x <= int(cx+r); x++ {
			if x < 0 || y < 0 || x >= S || y >= S {
				continue
			}
			d := math.Hypot(float64(x)-cx, float64(y)-cy)
			if d <= r {
				blend(img, x, y, c, 1)
			} else if d <= r+1.5 {
				blend(img, x, y, c, (r+1.5-d)/1.5)
			}
		}
	}
}

func blend(img *image.RGBA, x, y int, c color.RGBA, a float64) {
	if a >= 1 {
		img.Set(x, y, c)
		return
	}
	o := img.RGBAAt(x, y)
	img.Set(x, y, color.RGBA{
		uint8(float64(c.R)*a + float64(o.R)*(1-a)),
		uint8(float64(c.G)*a + float64(o.G)*(1-a)),
		uint8(float64(c.B)*a + float64(o.B)*(1-a)),
		0xff,
	})
}
