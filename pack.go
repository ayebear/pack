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

const OUT_DIR_PERMISSIONS = 0755

type ImageMap map[image.Point][]image.Image
type ImageChannel chan image.Image

type Options struct {
	inputDir  string
	outputDir string
	basename  string
	padding   int
}

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

func saveSpriteSheet(sheetWg *sync.WaitGroup, size image.Point, imageList []image.Image, options Options) {
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
	outPath := path.Join(options.outputDir, fmt.Sprintf("%s_%dx%d.png", options.basename, size.X, size.Y))
	saveImage(outPath, outImage)
}

func saveSpriteSheets(images ImageMap, options Options) {
	// Make sure output directory exists
	os.MkdirAll(options.outputDir, OUT_DIR_PERMISSIONS)

	// Save sheets in parallel
	sheetWg := sync.WaitGroup{}
	sheetWg.Add(len(images))
	for size, imageList := range images {
		go saveSpriteSheet(&sheetWg, size, imageList, options)
	}
	sheetWg.Wait()
}

func main() {
	// Register and parse command flags
	options := Options{}
	flag.StringVar(&options.inputDir, "in", "images", "Input directory path")
	flag.StringVar(&options.outputDir, "out", "images_out", "Output directory path")
	flag.StringVar(&options.basename, "name", "textures", "Basename to use for output filenames")
	flag.IntVar(&options.padding, "padding", 8, "Number of pixels to repeat around sprite edges (0 to disable)")
	flag.Parse()

	// Load images into map grouped by size
	images := loadImages(options.inputDir)

	// Write output images, one for each size
	saveSpriteSheets(images, options)

	// Write output metadata in json (TODO: figure out pixi.js compatibility)
}
