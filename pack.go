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
	baseName  string
	basePath  string
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
	sheetKey  string
}
type MetaSprite struct {
	name string
	pos  Position
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
	baseName := filepath.Base(path)
	name := strings.TrimSuffix(baseName, filepath.Ext(baseName))
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
	jsonPath := path.Join(options.outputDir, fmt.Sprintf("%s.json", options.baseName))
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

func drawSprite(outImage *image.RGBA, spriteChannel chan MetaSprite, i int, width int, padding int, img ImageDetails, size Size) {
	// Sprite position (not pixel position) in sprite sheet
	x, y := i%width, i/width
	// Top-left position of inner target
	pos := Position{padding + x*(size.W+padding), padding + y*(size.H+padding)}
	// Inner target contains actual sprite pixels
	innerTarget := image.Rect(pos.X, pos.Y, pos.X+size.W, pos.Y+size.H)
	// Outer target contains all repeated padding pixels and sprite pixels
	// outerTarget := image.Rect(pos.X, pos.Y, pos.X+size.W, pos.Y+size.H)

	// Draw sprite pixels and repeated padding pixels
	draw.Draw(outImage, innerTarget, img.image, image.Point{0, 0}, draw.Src)

	// Send sprite name and pos to channel
	spriteChannel <- MetaSprite{img.name, pos}
}

func saveSpriteSheet(sheetChannel chan SpriteSheet, size Size, imageList []ImageDetails, options Options) {
	// Sort image list by filename for deterministic output
	sort.Sort(byImagePath(imageList))

	// Create initial output image
	// TODO: Try to shrink vertical size if there would be unneeded space
	padding := options.padding
	imageCount := len(imageList)
	width := int(math.Ceil(math.Sqrt(float64(imageCount))))
	sheetSize := Size{width*size.W + padding*(width+1), width*size.H + padding*(width+1)}
	outImage := image.NewRGBA(image.Rect(0, 0, sheetSize.W, sheetSize.H))

	// Setup metadata for this sheet
	metaSheet := MetaSheet{sheetSize, Size{size.W, size.H}, make(map[string]Position)}

	// Copy all images to correct locations in output
	spriteChannel := make(chan MetaSprite)
	for i, img := range imageList {
		go drawSprite(outImage, spriteChannel, i, width, padding, img, size)
	}

	// Store sprite position metadata and wait for sprite threads to finish
	for i := 0; i < imageCount; i++ {
		spriteMeta := <-spriteChannel
		metaSheet.Sprites[spriteMeta.name] = spriteMeta.pos
	}

	// Save output image
	sheetFilename := fmt.Sprintf("%s_%dx%d.png", options.baseName, size.W, size.H)
	outPath := path.Join(options.outputDir, sheetFilename)
	saveImage(outPath, outImage)

	// Output key and metadata for sprite sheet
	sheetKey := sheetFilename
	if options.basePath != "" {
		sheetKey = path.Join(options.basePath, sheetFilename)
	}
	sheetChannel <- SpriteSheet{metaSheet, sheetKey}
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
		metadata[sheet.sheetKey] = sheet.metaSheet
	}
	return metadata
}

func main() {
	// Register and parse command flags
	options := Options{}
	flag.StringVar(&options.inputDir, "in", "images", "Input directory path containing individual sprites")
	flag.StringVar(&options.outputDir, "out", "images_out", "Output directory path for sheets and json")
	flag.StringVar(&options.baseName, "name", "textures", "Base filename to use for output filenames")
	flag.StringVar(&options.basePath, "path", "", "Base directory path to prepend to json metadata keys (leave empty for no parent directory in json sheet keys)")
	flag.IntVar(&options.padding, "padding", 8, "Number of pixels to repeat around sprite edges (0 to disable)")
	flag.Parse()

	// Load images into map grouped by size
	images := loadImages(options.inputDir)

	// Write output images, one for each size
	metadata := saveSpriteSheets(images, options)

	// Write output metadata in json
	saveMetadata(metadata, options)
}
