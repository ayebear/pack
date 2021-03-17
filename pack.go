package main

import (
	"flag"
	"fmt"
)

func main() {
	inputDir := flag.String("in", "images", "Input directory path")
	outputDir := flag.String("out", "images_out", "Output directory path")
	basename := flag.String("name", "textures", "Basename to use for output filenames")
	padding := flag.Int("padding", 8, "Number of pixels to repeat around sprite edges (0 to disable)")

	flag.Parse()

	fmt.Println("in:", *inputDir)
	fmt.Println("out:", *outputDir)
	fmt.Println("name:", *basename)
	fmt.Println("padding:", *padding)
}
