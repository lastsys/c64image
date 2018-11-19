package c64image

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/jpeg"
	"image/png"
	"math"
	"os"
)

// According to http://hitmen.c02.at/temp/palstuff/
var C64Colors = [16]color.RGBA{
	{0x00, 0x00, 0x00, 0xFF}, //  0 - black
	{0xFF, 0xFF, 0xFF, 0xFF}, //  1 - white
	{0x67, 0x37, 0x2B, 0xFF}, //  2 - red
	{0x6F, 0xA3, 0xB1, 0xFF}, //  3 - cyan
	{0x6F, 0x3C, 0x85, 0xFF}, //  4 - purple
	{0x58, 0x8C, 0x43, 0xFF}, //  5 - green
	{0x34, 0x28, 0x79, 0xFF}, //  6 - blue
	{0xB7, 0xC6, 0x6E, 0xFF}, //  7 - yellow
	{0x6F, 0x4F, 0x25, 0xFF}, //  8 - orange
	{0x42, 0x39, 0x00, 0xFF}, //  9 - brown
	{0x99, 0x66, 0x59, 0xFF}, // 10 - light red
	{0x43, 0x43, 0x43, 0xFF}, // 11 - dark grey
	{0x6B, 0x6B, 0x6B, 0xFF}, // 12 - grey
	{0x9A, 0xD1, 0x83, 0xFF}, // 13 - light green
	{0x6B, 0x5E, 0xB4, 0xFF}, // 14 - light blue
	{0x95, 0x95, 0x95, 0xFF}, // 15 - light grey
}

var UnsupportedStrideError = fmt.Errorf("unsupported stride")

type cielab struct {
	l float64
	a float64
	b float64
}

type xyz struct {
	x float64
	y float64
	z float64
}

const (
	C64Width  = 320
	C64Height = 200
)

func LoadImage(filename string) (*image.RGBA, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	img, _, err := image.Decode(file)
	if err != nil {
		return nil, err
	}
	rgba := image.NewRGBA(img.Bounds())
	if rgba.Stride != rgba.Rect.Size().X*4 {
		return nil, UnsupportedStrideError
	}
	draw.Draw(rgba, rgba.Bounds(), img, image.Point{0, 0}, draw.Src)
	return rgba, nil
}

func SaveImage(img *image.RGBA, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	png.Encode(file, img)
	return nil
}

func ConvertImage(img *image.RGBA) *image.RGBA {
	aspectRatio := float64(img.Rect.Size().X) / float64(img.Rect.Size().Y)

	targetWidth := 320
	targetHeight := int(math.Ceil(float64(targetWidth) / aspectRatio))

	targetImage := image.NewRGBA(image.Rect(0, 0, targetWidth, targetHeight))

	blockWidth := int(float64(img.Rect.Size().X) / float64(targetWidth/2))
	blockHeight := int(float64(img.Rect.Size().Y) / float64(targetHeight))

	for j := 0; j < targetHeight; j++ {
		for i := 0; i <= targetWidth; i++ {
			colorEstimate := meanBlockColor(img,
				image.Rect(i*blockWidth, j*blockHeight, (i+1)*blockWidth-1, (j+1)*blockHeight-1))
			ci := closestC64Color(colorEstimate)
			targetImage.SetRGBA(i*2, j, C64Colors[ci])
			targetImage.SetRGBA(i*2+1, j, C64Colors[ci])
		}
	}

	return targetImage
}

// Calculate mean color of image block.
func meanBlockColor(img *image.RGBA, rect image.Rectangle) cielab {
	avglab := cielab{0, 0, 0}
	for x := rect.Min.X; x < rect.Max.X; x++ {
		for y := rect.Min.Y; y < rect.Max.Y; y++ {
			lab := convertRGBAtoCIELAB(img.RGBAAt(x, y))
			avglab.l += lab.l
			avglab.a += lab.a
			avglab.b += lab.b
		}
	}
	pixelCount := float64((rect.Max.X - rect.Min.X) * (rect.Max.Y - rect.Min.Y))
	avglab.l /= pixelCount
	avglab.a /= pixelCount
	avglab.b /= pixelCount
	return avglab
}

// Find index of closest color in C64Colors.
func closestC64Color(color cielab) int {
	bestIndex := 0
	bestDistance := math.Inf(1)
	for i, c64Color := range C64Colors {
		lab2 := convertRGBAtoCIELAB(c64Color)
		if distance := cie94distance(color, lab2); distance < bestDistance {
			bestIndex = i
			bestDistance = distance
		}
	}
	return bestIndex
}

func colorDistance(color1 color.RGBA, color2 color.RGBA) float64 {
	return math.Pow(float64(color1.R)-float64(color2.R), 2) +
		math.Pow(float64(color1.G)-float64(color2.G), 2) +
		math.Pow(float64(color1.B)-float64(color2.B), 2)
}

func convertRGBAtoXYZ(rgba color.RGBA) xyz {
	return xyz{
		x: 0.412453*float64(rgba.R) + 0.357580*float64(rgba.G) + 0.180423*float64(rgba.B),
		y: 0.212671*float64(rgba.R) + 0.715160*float64(rgba.G) + 0.072169*float64(rgba.B),
		z: 0.019334*float64(rgba.R) + 0.119193*float64(rgba.G) + 0.950227*float64(rgba.B),
	}
}

func convertRGBAtoCIELAB(rgba color.RGBA) cielab {
	xyz := convertRGBAtoXYZ(rgba)
	// Using illuminant D65 with normalization for Y as reference white point.
	Xn := 95.047
	Yn := 100.0
	Zn := 108.883

	lf := func(t float64) float64 {
		if t > 0.008856 {
			return 116.0*math.Pow(t, 1.0/3.0) - 16.0
		}
		return 903.3 * t
	}

	f := func(t float64) float64 {
		if t > 0.008856 {
			return math.Pow(t, 1.0/3.0)
		}
		return 7.787*t + 16.0/116.0
	}

	return cielab{
		l: lf(xyz.y / Yn),
		a: 500.0 * (f(xyz.x/Xn) - f(xyz.y/Yn)),
		b: 200.0 * (f(xyz.y/Yn) - f(xyz.z/Zn)),
	}
}

func cie76distance(c1 cielab, c2 cielab) float64 {
	return math.Pow(c2.l-c1.l, 2.0) + math.Pow(c2.a-c1.a, 2.0) + math.Pow(c2.b-c2.b, 2.0)
}

func cie94distance(col1 cielab, col2 cielab) float64 {
	dL := col1.l - col2.l
	c1 := math.Sqrt(col1.a*col1.a + col1.b*col1.b)
	c2 := math.Sqrt(col2.a*col2.a + col2.b*col2.b)
	da := col1.a - col2.a
	db := col2.b - col2.b
	dCab := c1 - c2
	dHab := math.Sqrt(da*da + db*db + dCab*dCab)
	kL := 1.0
	kC := 1.0
	kH := 1.0
	k1 := 0.045
	k2 := 0.015
	sL := 1.0
	sC := 1.0 + k1*c1
	sH := 1.0 + k2*c1
	return math.Pow(dL/(kL*sL), 2) + math.Pow(dCab/(kC*sC), 2) + math.Pow(dHab/(kH*sH), 2)
}
