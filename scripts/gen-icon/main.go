// Generates a 1024x1024 source icon for MeshManager: a dark rounded tile with
// a stylized "mesh" of connected nodes in the accent color.
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
	light := color.RGBA{0xe4, 0xe9, 0xf2, 0xff}

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

	// Mesh nodes (normalized positions within the tile).
	nodes := [][2]float64{
		{0.30, 0.30}, {0.70, 0.26}, {0.50, 0.50},
		{0.26, 0.70}, {0.74, 0.72},
	}
	edges := [][2]int{{0, 1}, {0, 2}, {1, 2}, {2, 3}, {2, 4}, {3, 4}, {0, 3}, {1, 4}}

	pt := func(n [2]float64) (float64, float64) { return n[0] * S, n[1] * S }

	for _, e := range edges {
		x1, y1 := pt(nodes[e[0]])
		x2, y2 := pt(nodes[e[1]])
		drawLine(img, x1, y1, x2, y2, 14, accent)
	}
	for i, n := range nodes {
		cx, cy := pt(n)
		r := 56.0
		c := accent
		if i == 2 {
			r = 74
			c = light
		}
		drawDisc(img, cx, cy, r, c)
	}

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

func drawLine(img *image.RGBA, x1, y1, x2, y2, w float64, c color.RGBA) {
	steps := int(math.Hypot(x2-x1, y2-y1))
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		drawDisc(img, x1+(x2-x1)*t, y1+(y2-y1)*t, w/2, c)
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
