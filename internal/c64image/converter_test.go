package c64image

import (
	"image/color"
	"math"
	"testing"
)

func TestColorDistance(t *testing.T) {
	c1 := color.RGBA{10, 20, 30, 255}
	c2 := color.RGBA{200, 190, 180, 255}

	d := rgbDistance(c1, c2)

	if math.Abs(d-87500.0) > 0.1 {
		t.Errorf("color distance broken, got %v but expected %v", d, 87500.0)
	}
}
