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

func almostZero(x float64) bool {
	return math.Abs(x) < 1e-8
}

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

type rgb struct {
	r float64
	g float64
	b float64
}

const (
	C64Width  = 320
	C64Height = 200
)

type Method int

const (
	RGBMethod Method = iota
	CIE76
	CIE94
	CIE2000
)

const (
	deg2rad = math.Pi / 180.
	rad2deg = 180. / math.Pi
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

func ConvertImage(img *image.RGBA, method Method, returnChannel chan *image.RGBA) {
	aspectRatio := float64(img.Rect.Size().X) / float64(img.Rect.Size().Y)

	targetWidth := 320
	targetHeight := int(math.Ceil(float64(targetWidth) / aspectRatio))

	targetImage := image.NewRGBA(image.Rect(0, 0, targetWidth, targetHeight))

	blockWidth := int(float64(img.Rect.Size().X) / float64(targetWidth/2))
	blockHeight := int(float64(img.Rect.Size().Y) / float64(targetHeight))

	for j := 0; j < targetHeight; j++ {
		for i := 0; i <= targetWidth; i++ {
			colorEstimate, rgbEstimate := meanBlockColor(img,
				image.Rect(i*blockWidth, j*blockHeight, (i+1)*blockWidth-1, (j+1)*blockHeight-1))
			ci := closestC64Color(colorEstimate, rgbEstimate, method)
			targetImage.SetRGBA(i*2, j, C64Colors[ci])
			targetImage.SetRGBA(i*2+1, j, C64Colors[ci])
		}
	}

	returnChannel <- targetImage
}

// Calculate mean color of image block.
func meanBlockColor(img *image.RGBA, rect image.Rectangle) (cielab, color.RGBA) {
	avglab := cielab{0, 0, 0}
	avgrgb := rgb{0, 0, 0}
	var rgbColor color.RGBA
	for x := rect.Min.X; x < rect.Max.X; x++ {
		for y := rect.Min.Y; y < rect.Max.Y; y++ {
			rgbColor = img.RGBAAt(x, y)
			lab := convertRGBAtoCIELAB(rgbColor)
			avglab.l += lab.l
			avglab.a += lab.a
			avglab.b += lab.b

			avgrgb.r += float64(rgbColor.R)
			avgrgb.g += float64(rgbColor.G)
			avgrgb.b += float64(rgbColor.B)
		}
	}
	pixelCount := float64((rect.Max.X - rect.Min.X) * (rect.Max.Y - rect.Min.Y))

	avglab.l /= pixelCount
	avglab.a /= pixelCount
	avglab.b /= pixelCount

	avgrgb.r /= pixelCount
	avgrgb.g /= pixelCount
	avgrgb.b /= pixelCount

	avgrgbColor := color.RGBA{uint8(avgrgb.r), uint8(avgrgb.g), uint8(avgrgb.b), 255}

	return avglab, avgrgbColor
}

// Find index of closest color in C64Colors.
func closestC64Color(color cielab, rgbColor color.RGBA, method Method) int {
	bestIndex := 0
	bestDistance := math.Inf(1)
	for i, c64Color := range C64Colors {
		lab2 := convertRGBAtoCIELAB(c64Color)

		var deltaE float64
		switch method {
		case RGBMethod:
			deltaE = rgbDistance(rgbColor, c64Color)
		case CIE76:
			deltaE = cie76distance(color, lab2)
		case CIE94:
			deltaE = cie94distance(color, lab2)
		case CIE2000:
			deltaE = cie2000distance(color, lab2)
		}

		if deltaE < bestDistance {
			bestIndex = i
			bestDistance = deltaE
		}
	}
	return bestIndex
}

func rgbDistance(color1 color.RGBA, color2 color.RGBA) float64 {
	return math.Pow(float64(color1.R)-float64(color2.R), 2) +
		math.Pow(float64(color1.G)-float64(color2.G), 2) +
		math.Pow(float64(color1.B)-float64(color2.B), 2)
}

func convertRGBAtoXYZ(rgba color.RGBA) xyz {
	r := float64(rgba.R) / 255.0
	g := float64(rgba.G) / 255.0
	b := float64(rgba.B) / 255.0

	if r > 0.04045 {
		r = math.Pow((r+0.055)/1.055, 2.4)
	} else {
		r = r / 12.92
	}

	if g > 0.04045 {
		g = math.Pow((g+0.055)/1.055, 2.4)
	} else {
		g = g / 12.92
	}

	if b > 0.04045 {
		b = math.Pow((b+0.055)/1.055, 2.4)
	} else {
		b = b / 12.92
	}

	r *= 100.0
	g *= 100.0
	b *= 100.0

	return xyz{
		x: 0.4124564*r + 0.3575761*g + 0.1804375*b,
		y: 0.2126729*r + 0.7151522*g + 0.0721750*b,
		z: 0.0193339*r + 0.1191920*g + 0.9503041*b,
	}
}

func convertRGBAtoCIELAB(rgba color.RGBA) cielab {
	xyz := convertRGBAtoXYZ(rgba)
	// Using illuminant D65 with normalization for Y as reference white point.
	Xn := 95.047
	Yn := 100.0
	Zn := 108.883

	f := func(t float64) float64 {
		if t > math.Pow(24./116., 3.) {
			return math.Pow(t, 1./3.)
		}
		return (841./108.)*t + 16./116.
	}

	return cielab{
		l: 116.*f(xyz.y/Yn) - 16.,
		a: 500. * (f(xyz.x/Xn) - f(xyz.y/Yn)),
		b: 200. * (f(xyz.y/Yn) - f(xyz.z/Zn)),
	}
}

func cie76distance(c1 cielab, c2 cielab) float64 {
	return math.Pow(c2.l-c1.l, 2.0) + math.Pow(c2.a-c1.a, 2.0) + math.Pow(c2.b-c2.b, 2.0)
}

func cie94distance(col1 cielab, col2 cielab) float64 {
	xC1 := math.Sqrt(col1.a*col1.a + col1.b*col1.b)
	xC2 := math.Sqrt(col2.a*col2.a + col2.b*col2.b)
	xDL := col2.l - col1.l
	xDC := xC2 - xC1
	xDE := math.Sqrt(math.Pow(col1.l-col2.l, 2.) +
		math.Pow(col1.a-col2.a, 2.) +
		math.Pow(col1.b-col2.b, 2.))
	xDH := xDE*xDE - xDL*xDL - xDC*xDC
	if xDH > 0. {
		xDH = math.Sqrt(xDH)
	} else {
		xDH = 0.
	}
	xSC := 1. + (0.045 * xC1)
	xSH := 1. + (0.015 * xC1)

	xDL /= 1.
	xDC /= 1. * xSC
	xDH /= 1. * xSH

	return xDL*xDL + xDC*xDC + xDH*xDH
}

func cielab2hue(a, b float64) float64 {
	return math.Atan2(b, a) * rad2deg
}

func cie2000distance(col1 cielab, col2 cielab) float64 {
	xC1 := math.Sqrt(col1.a*col1.a + col1.b*col1.b)
	xC2 := math.Sqrt(col2.a*col2.a + col2.b*col2.b)
	xCX := (xC1 + xC2) / 2.
	xGX := .5 * (1. - math.Sqrt(math.Pow(xCX, 7.)/(math.Pow(xCX, 7.)+math.Pow(25., 7.))))
	xNN := (1. + xGX) * col1.a
	xC1 = math.Sqrt(xNN*xNN + col1.b*col1.b)
	xH1 := cielab2hue(xNN, col1.b)
	xNN = (1. + xGX) * col2.a
	xC2 = math.Sqrt(xNN*xNN + col2.b*col2.b)
	xH2 := cielab2hue(xNN, col2.b)
	xDL := col2.l - col1.l
	xDC := xC2 - xC1
	var xDH float64
	if almostZero(xC1 * xC2) {
		xDH = 0.
	} else {
		xNN = xH2 - xH1 // round to 12 decimal places???
		if math.Abs(xNN) <= 180. {
			xDH = xH2 - xH1
		} else {
			if xNN > 180. {
				xDH = xH2 - xH1 - 360.
			} else {
				xDH = xH2 - xH1 + 360.
			}
		}
	}
	xDH = 2. * math.Sqrt(xC1*xC2) * math.Sin(deg2rad*(xDH/2.))
	xLX := (col1.l + col2.l) / 2.
	xCY := (xC1 + xC2) / 2.
	var xHX float64
	if almostZero(xC1 * xC2) {
		xHX = xH1 + xH2
	} else {
		xNN = math.Abs(xH1 - xH2) // round to 12 decimal places???
		if xNN > 180. {
			if (xH2 + xH1) < 360. {
				xHX = xH1 + xH2 + 360.
			} else {
				xHX = xH1 + xH2 - 360.
			}
		} else {
			xHX = xH1 + xH2
		}
		xHX /= 2.
	}
	xTX := 1. - 0.17*math.Cos(deg2rad*(xHX-30.)) +
		0.24*math.Cos(deg2rad*(2.*xHX)) +
		0.32*math.Cos(deg2rad*(3.*xHX+6.)) -
		0.20*math.Cos(deg2rad*(4.*xHX-63.))
	xPH := 30. * math.Exp(-((xHX-275.)/25.)*((xHX-275.)/25.))
	xRC := 2. * math.Sqrt(math.Pow(xCY, 7.)/(math.Pow(xCY, 7.)+math.Pow(25., 7.)))
	xSL := 1. + (0.015*(xLX-50.)*(xLX-50.))/math.Sqrt(20.+(xLX-50.)*(xLX-50.))
	xSC := 1. + 0.045*xCY
	xSH := 1. + 0.015*xCY*xTX
	xRT := -math.Sin(deg2rad*2.*xPH) * xRC
	xDL /= xSL
	xDC /= xSC
	xDH /= xSH

	return xDL*xDL + xDC*xDC + xDH*xDH + xRT*xDC*xDH
}
