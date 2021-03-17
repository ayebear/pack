package main

import (
	"flag"
	"fmt"
	"image"
	_ "image/png"
	"os"
	"path/filepath"
)

type ImageMap map[image.Point][]image.Image
type ImageChannel chan image.Image

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func getImageFromFilePath(filePath string) image.Image {
	f, err := os.Open(filePath)
	check(err)
	defer f.Close()
	img, _, err := image.Decode(f)
	check(err)
	return img
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
			imageChannel <- getImageFromFilePath(p)
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

func main() {
	// Register and parse command flags
	inputDir := flag.String("in", "images", "Input directory path")
	outputDir := flag.String("out", "images_out", "Output directory path")
	basename := flag.String("name", "textures", "Basename to use for output filenames")
	padding := flag.Int("padding", 8, "Number of pixels to repeat around sprite edges (0 to disable)")
	flag.Parse()

	// TODO: Remove debug
	fmt.Println("in:", *inputDir)
	fmt.Println("out:", *outputDir)
	fmt.Println("name:", *basename)
	fmt.Println("padding:", *padding)

	// Load images into map grouped by size
	images := loadImages(*inputDir)
	fmt.Println(images)

	// Write output images, one for each size
	// saveImages(&images)

	// Write output metadata in json (TODO: figure out pixi.js compatibility)
}
