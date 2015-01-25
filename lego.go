// Copyright 2014 Leonardo "Bubble" Mesquita
package lego

import (
	"fmt"
	"github.com/nfnt/resize"
	"image"
	"image/color"
	"image/draw"
)

type Color struct {
	name  string
	color color.Color
}

var (
	// Names and values retrieved from:
	//   http://www.peeron.com/cgi-bin/invcgis/colorguide.cgi
	// Selected colors from http://shop.lego.com that are available for
	// 1x1 bricks, so any images are doable.
	WHITE                  = Color{"White (#1)", color.NRGBA{242, 243, 242, 255}}
	BRIGHT_RED             = Color{"Bright red (#21)", color.NRGBA{196, 40, 27, 255}}
	BRIGHT_BLUE            = Color{"Bright blue (#23)", color.NRGBA{13, 105, 171, 255}}
	BLACK                  = Color{"Black (#26)", color.NRGBA{27, 42, 52, 255}}
	DARK_GREEN             = Color{"Dark green (#28)", color.NRGBA{40, 127, 70, 255}}
	BRIGHT_YELLOW          = Color{"Bright yellow (#24)", color.NRGBA{245, 205, 47, 255}}
	BRICK_YELLOW           = Color{"Brick yellow (#5)", color.NRGBA{215, 197, 153, 255}}
	BRIGHT_ORANGE          = Color{"Bright orange (#106)", color.NRGBA{218, 133, 64, 255}}
	MEDIUM_BLUE            = Color{"Medium blue (#102)", color.NRGBA{110, 153, 201, 255}}
	DARK_STONE_GREY        = Color{"Dark stone grey (#199)", color.NRGBA{99, 95, 97, 255}}
	REDDISH_BROWN          = Color{"Reddish brown (#192)", color.NRGBA{105, 64, 39, 255}}
	MEDIUM_STONE_GREY      = Color{"Medium stone grey (#194)", color.NRGBA{163, 162, 164, 255}}
	BRIGHT_YELLOWISH_GREEN = Color{"Bright yellowish green (#119)", color.NRGBA{164, 189, 70, 255}}
	LIGHT_PURPLE           = Color{"Light purple (#222)", color.NRGBA{228, 173, 200, 255}}
	BRIGHT_REDDISH_VIOLET  = Color{"Bright reddish violet (#124)", color.NRGBA{146, 57, 120, 255}}
)

func (c *Color) Name() string {
	return c.name
}

func (c *Color) Color() color.Color {
	return c.color
}

type Brick struct {
	Size  image.Point
	Color Color
}

func generateBricks(shapes []image.Point, colors ...Color) []*Brick {
	var result []*Brick
	for _, color := range colors {
		for _, shape := range shapes {
			result = append(result, &Brick{shape, color})
		}
	}
	return result
}

var (
	basicShapes = []image.Point{
		{1, 1}, {1, 2}, {1, 4}, {2, 2}, {2, 4},
	}
	BASIC_BRICKS = generateBricks(basicShapes, WHITE, BRIGHT_RED, BRIGHT_BLUE,
		BLACK, DARK_GREEN, BRIGHT_YELLOW, BRICK_YELLOW, BRIGHT_ORANGE,
	)
	ADVANCED_BRICKS = append(
		generateBricks(basicShapes, DARK_STONE_GREY, REDDISH_BROWN,
			MEDIUM_STONE_GREY, BRIGHT_YELLOWISH_GREEN, LIGHT_PURPLE),
		append(generateBricks([]image.Point{{1, 1}, {1, 2}, {1, 4}}, MEDIUM_BLUE),
			generateBricks([]image.Point{{1, 1}, {1, 2}}, BRIGHT_REDDISH_VIOLET)...,
		)...,
	)
	ALL_BRICKS = append(BASIC_BRICKS, ADVANCED_BRICKS...)
)

func (b Brick) String() string {
	return fmt.Sprintf("%dx%d %s", b.Size.X, b.Size.Y, b.Color.name)
}

func (b Brick) canonical() Brick {
	if b.Size.X <= b.Size.Y {
		return b
	}
	return Brick{image.Point{b.Size.Y, b.Size.X}, b.Color}
}

type Panel struct {
	bricks map[image.Point]*Brick
	bounds image.Rectangle
}

type Options struct {
	Width  uint
	Bricks []*Brick
	Dither bool
}

type helper struct {
	visited map[image.Point]bool
	panel   *Panel
	bricks  map[Brick]bool
	img     image.Image
}

func newHelper(bricks []*Brick, img image.Image, p *Panel) *helper {
	ret := &helper{
		visited: make(map[image.Point]bool),
		panel:   p,
		bricks:  make(map[Brick]bool),
		img:     img,
	}
	for _, brick := range bricks {
		ret.bricks[*brick] = true
	}
	return ret
}

func (h *helper) fit(p image.Point, brick Brick) bool {
	for y := 0; y < brick.Size.Y; y++ {
		for x := 0; x < brick.Size.X; x++ {
			pt := p.Add(image.Point{x, y})
			if h.visited[pt] {
				return false
			}
			if h.img.At(pt.X, pt.Y) != brick.Color.color {
				return false
			}
		}
	}
	return true
}

func (h *helper) placeBrick(p image.Point, color Color) {
	if h.visited[p] {
		return
	}
	for i := range basicShapes {
		shape := basicShapes[len(basicShapes)-1-i]
		brick := Brick{shape, color}
		if !h.bricks[brick] {
			continue
		}
		if !h.fit(p, brick) {
			if shape.X == shape.Y {
				continue
			}
			brick = Brick{image.Point{shape.Y, shape.X}, color}
			if !h.fit(p, brick) {
				continue
			}
		}
		for y := 0; y < brick.Size.Y; y++ {
			for x := 0; x < brick.Size.X; x++ {
				h.visited[p.Add(image.Point{x, y})] = true
			}
		}
		h.panel.bricks[p] = &brick
		return
	}
	panic("Impossible fit")
}

func NewPanel(img image.Image, opt *Options) *Panel {
	scale := float64(opt.Width) / float64(img.Bounds().Dx())
	height := uint(scale * float64(img.Bounds().Dy()))

	var palette color.Palette
	m := make(map[color.Color]Color)
	for _, brick := range opt.Bricks {
		if _, ok := m[brick.Color.color]; !ok {
			m[brick.Color.color] = brick.Color
			palette = append(palette, brick.Color.color)
		}
	}

	src := resize.Resize(opt.Width, height, img, resize.Lanczos3)
	dst := image.NewPaletted(src.Bounds(), palette)
	if opt.Dither {
		draw.FloydSteinberg.Draw(dst, dst.Bounds(), src, src.Bounds().Min)
	} else {
		draw.Draw(dst, dst.Bounds(), src, src.Bounds().Min, draw.Src)
	}
	ret := &Panel{make(map[image.Point]*Brick), dst.Bounds()}
	helper := newHelper(opt.Bricks, dst, ret)
	for y := dst.Bounds().Min.Y; y < dst.Bounds().Max.Y; y++ {
		for x := dst.Bounds().Min.X; x < dst.Bounds().Max.X; x++ {
			helper.placeBrick(image.Point{x, y}, m[dst.At(x, y)])
		}
	}
	return ret
}

func (p *Panel) Draw(scale int, outline bool) image.Image {
	out := image.NewNRGBA(image.Rectangle{image.ZP, p.bounds.Size().Mul(scale)})
	draw.Draw(out, out.Bounds(), &image.Uniform{color.White}, image.ZP, draw.Src)
	for pos, brick := range p.bricks {
		min := pos.Mul(scale)
		max := min.Add(brick.Size.Mul(scale))
		if outline {
			draw.Draw(out, image.Rectangle{min, max}, &image.Uniform{color.NRGBA{0, 0, 0, 255}},
				image.ZP, draw.Src)
			min = min.Add(image.Point{1, 1})
			max = max.Sub(image.Point{1, 1})
			draw.Draw(out, image.Rectangle{min, max}, &image.Uniform{color.NRGBA{255, 255, 255, 255}},
				image.ZP, draw.Src)
			min = min.Add(image.Point{1, 1})
			max = max.Sub(image.Point{1, 1})
		}
		draw.Draw(out, image.Rectangle{min, max}, &image.Uniform{brick.Color.color},
			image.ZP, draw.Src)
	}
	return out
}

func (p *Panel) Size() image.Point {
	return p.bounds.Size()
}

func (p *Panel) CountBricks() map[Brick]int {
	result := make(map[Brick]int)
	for _, brick := range p.bricks {
		result[brick.canonical()] += 1
	}
	return result
}
