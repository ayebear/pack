package main

import (
	"encoding/json"
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
	"sort"
	"strings"
)

const OUT_DIR_PERMISSIONS = 0755

type Size struct {
	W int `json:"w"`
	H int `json:"h"`
}
type Position struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type ImageDetails struct {
	image image.Image
	path  string
	name  string
}
type ImageMap map[Size][]ImageDetails
type ImageChannel chan ImageDetails
type byImagePath []ImageDetails

type Options struct {
	inputDir  string
	outputDir string
	basename  string
	padding   int
}

type MetaSheet struct {
	SheetSize  Size                `json:"sheetSize"`
	SpriteSize Size                `json:"spriteSize"`
	Sprites    map[string]Position `json:"sprites"`
}
type MetaRoot map[string]MetaSheet
type SpriteSheet struct {
	metaSheet MetaSheet
	sheetName string
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func loadImage(path string) ImageDetails {
	f, err := os.Open(path)
	check(err)
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		fmt.Println(path, "could not be decoded.")
		panic(err)
	}
	// Get base name without extension for use as sprite key
	basename := filepath.Base(path)
	name := strings.TrimSuffix(basename, filepath.Ext(basename))
	return ImageDetails{img, path, name}
}

func saveImage(path string, img image.Image) {
	f, err := os.Create(path)
	check(err)
	err = png.Encode(f, img)
	check(err)
	err = f.Close()
	check(err)
}

func writeBytesToFile(data []byte, path string) {
	f, err := os.Create(path)
	check(err)
	_, err = f.Write(data)
	check(err)
	err = f.Close()
	check(err)
}

func saveMetadata(metadata MetaRoot, options Options) {
	data, err := json.Marshal(metadata)
	check(err)
	jsonPath := path.Join(options.outputDir, fmt.Sprintf("%s.json", options.basename))
	writeBytesToFile(data, jsonPath)
}

// For sorting images by path
func (s byImagePath) Len() int {
	return len(s)
}
func (s byImagePath) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s byImagePath) Less(i, j int) bool {
	return s[i].path < s[j].path
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
		size := img.image.Bounds().Max
		key := Size{size.X, size.Y}
		images[key] = append(images[key], img)
	}
	return images
}

func saveSpriteSheet(sheetChannel chan SpriteSheet, size Size, imageList []ImageDetails, options Options) {
	// Sort image list by filename for deterministic output
	sort.Sort(byImagePath(imageList))

	// Create initial output image
	// TODO: Try to shrink vertical size if there would be unneeded space
	width := int(math.Ceil(math.Sqrt(float64(len(imageList)))))
	sheetSize := Size{width * size.W, width * size.H}
	outImage := image.NewRGBA(image.Rect(0, 0, sheetSize.W, sheetSize.H))

	// Setup metadata for this sheet
	metaSheet := MetaSheet{Size{size.W, size.H}, sheetSize, make(map[string]Position)}

	// Copy all images to correct locations in output
	for i, img := range imageList {
		// TODO: Move to separate func, mt with go
		// For every pixel in the output, map to a position in input
		// Use a simple clipping alg where top-left is shifted by (padding, padding)
		// Clip to either corner or edge when out of bounds
		x, y := i%width, i/width
		pos := Position{x * size.W, y * size.H}
		target := image.Rect(pos.X, pos.Y, pos.X+size.W, pos.Y+size.H)
		draw.Draw(outImage, target, img.image, image.Point{0, 0}, draw.Src)

		// Store sprite metadata
		metaSheet.Sprites[img.name] = pos
	}

	// Save output image
	sheetName := fmt.Sprintf("%s_%dx%d.png", options.basename, size.W, size.H)
	outPath := path.Join(options.outputDir, sheetName)
	saveImage(outPath, outImage)

	// Output metadata
	sheetChannel <- SpriteSheet{metaSheet, sheetName}
}

func saveSpriteSheets(images ImageMap, options Options) MetaRoot {
	// Make sure output directory exists
	os.MkdirAll(options.outputDir, OUT_DIR_PERMISSIONS)

	// Save sheets in parallel
	n := len(images)
	sheetChannel := make(chan SpriteSheet, n)
	for size, imageList := range images {
		go saveSpriteSheet(sheetChannel, size, imageList, options)
	}

	// Store sheet metadata into main metadata
	metadata := MetaRoot{}
	for i := 0; i < n; i++ {
		sheet := <-sheetChannel
		metadata[sheet.sheetName] = sheet.metaSheet
	}
	return metadata
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
	metadata := saveSpriteSheets(images, options)

	// Write output metadata in json
	saveMetadata(metadata, options)
}
