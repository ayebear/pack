package main

import (
	"flag"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	_ "image/png"
	"math"
	"os"
	"path"
	"path/filepath"
	"sync"
)

type ImageMap map[image.Point][]image.Image
type ImageChannel chan image.Image

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func loadImage(filePath string) image.Image {
	f, err := os.Open(filePath)
	check(err)
	defer f.Close()
	img, _, err := image.Decode(f)
	check(err)
	return img
}

func saveImage(filePath string, img image.Image) {
	f, err := os.Create(filePath)
	check(err)
	err = png.Encode(f, img)
	check(err)
	err = f.Close()
	check(err)
}

func loadImages(inputDir string) ImageMap {
	// Load images into channel
	imageChannel := make(ImageChannel)
	n := 0
	err := filepath.Walk(inputDir, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		n++
		go func() {
			imageChannel <- loadImage(p)
		}()
		return nil
	})
	check(err)

	// Group images by size into map
	images := make(ImageMap)
	for i := 0; i < n; i++ {
		img := <-imageChannel
		key := img.Bounds().Max
		images[key] = append(images[key], img)
	}
	return images
}

func saveSpriteSheet(sheetWg *sync.WaitGroup, size image.Point, imageList []image.Image, outputDir string, basename string, padding int) {
	defer sheetWg.Done()

	// Create initial output image
	width := int(math.Ceil(math.Sqrt(float64(len(imageList)))))
	outSize := image.Rect(0, 0, width*size.X, width*size.Y)
	outImage := image.NewRGBA(outSize)

	// Copy all images to correct locations in output
	for i, img := range imageList {
		x, y := i%width, i/width
		target := image.Rect(x*size.X, y*size.Y, x*size.X+size.X, y*size.Y+size.Y)
		draw.Draw(outImage, target, img, image.Point{0, 0}, draw.Src)
	}

	// Save output image
	outPath := path.Join(outputDir, fmt.Sprintf("%s_%dx%d.png", basename, size.X, size.Y))
	saveImage(outPath, outImage)
}

func saveSpriteSheets(images ImageMap, outputDir string, basename string, padding int) {
	// Make sure output directory exists
	// TODO: Magic number
	os.MkdirAll(outputDir, 0755)

	// Save sheets in parallel
	var sheetWg sync.WaitGroup
	sheetWg.Add(len(images))
	for size, imageList := range images {
		go saveSpriteSheet(&sheetWg, size, imageList, outputDir, basename, padding)
	}
	sheetWg.Wait()
}

func main() {
	// TODO: Store in options object
	// Register and parse command flags
	inputDir := flag.String("in", "images", "Input directory path")
	outputDir := flag.String("out", "images_out", "Output directory path")
	basename := flag.String("name", "textures", "Basename to use for output filenames")
	padding := flag.Int("padding", 8, "Number of pixels to repeat around sprite edges (0 to disable)")
	flag.Parse()

	// Load images into map grouped by size
	images := loadImages(*inputDir)

	// Write output images, one for each size
	saveSpriteSheets(images, *outputDir, *basename, *padding)

	// Write output metadata in json (TODO: figure out pixi.js compatibility)
}
