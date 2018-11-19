package main

import (
	"github.com/lastsys/c64image/internal/c64image"
	"io/ioutil"
	"log"
	"strings"
)

func main() {
	files, err := ioutil.ReadDir("./")
	if err != nil {
		panic(err)
	}

	for _, f := range files {

		if strings.HasSuffix(f.Name(), ".jpg") {
			originalImage, err := c64image.LoadImage(f.Name())
			if err != nil {
				panic(err)
			}
			baseFilename := strings.Trim(f.Name(), ".jpg")
			log.Printf("Processing %v\n", baseFilename)
			c64Image := c64image.ConvertImage(originalImage)
			targetFile := "c64_" + baseFilename + ".png"
			c64image.SaveImage(c64Image, targetFile)
		}
	}
}
