package main

import (
	"github.com/lastsys/c64image/internal/c64image"
	"image"
	"io/ioutil"
	"log"
	"strings"
)

func main() {
	files, err := ioutil.ReadDir("./")
	if err != nil {
		panic(err)
	}

	rgbChannel := make(chan *image.RGBA)
	cie76Channel := make(chan *image.RGBA)
	cie94Channel := make(chan *image.RGBA)
	cie2000Channel := make(chan *image.RGBA)

	for _, f := range files {

		if strings.HasSuffix(f.Name(), ".jpg") {
			originalImage, err := c64image.LoadImage(f.Name())
			if err != nil {
				panic(err)
			}
			baseFilename := strings.TrimSuffix(f.Name(), ".jpg")
			log.Printf("Processing %v\n", baseFilename)
			go c64image.ConvertImage(originalImage, c64image.RGBMethod, rgbChannel)
			go c64image.ConvertImage(originalImage, c64image.CIE76, cie76Channel)
			go c64image.ConvertImage(originalImage, c64image.CIE94, cie94Channel)
			go c64image.ConvertImage(originalImage, c64image.CIE2000, cie2000Channel)

			log.Print("Saving\n")
			c64image.SaveImage(<-rgbChannel, "c64_"+baseFilename+"_RGB.png")
			c64image.SaveImage(<-cie76Channel, "c64_"+baseFilename+"_CIE76.png")
			c64image.SaveImage(<-cie94Channel, "c64_"+baseFilename+"_CIE94.png")
			c64image.SaveImage(<-cie2000Channel, "c64_"+baseFilename+"_CIE2000.png")
			log.Print("Done.\n")
			log.Print("-----------------------------\n")
		}
	}
}
