package main

import (
	"flag"
	"fmt"
	"os"
	imageHandling "pixf/internal/toolset"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

func printHelp() {
	fmt.Println(`Usage: pixf [OPTIONS] <pdf-file> [format]

A tool for working with PDF files - unlock PDFs and extract images.

Arguments:
  pdf-file     Path to the PDF file to process (required)
  format       Image output format (optional, defaults to 'original')
               Supported formats: original, png, webp

Options:
  -h, --help           Show this help message
  --unlock-only        Only unlock the PDF, do not extract images
  --extract-only       Only extract images, do not unlock the PDF first

Format Options:
  original    Extract images using PDF's native format (default)
  png         Extract as PNG with transparency support
  webp        Extract as WebP with transparency support

Examples:
  pixf document.pdf                    # Unlock and extract images (original format)
  pixf document.pdf png                # Unlock and extract as PNG
  pixf --unlock-only document.pdf      # Only unlock the PDF
  pixf --extract-only document.pdf     # Only extract images from PDF
  pixf -h                              # Show this help message`)
}

func main() {
	// Define flags
	helpFlag := flag.Bool("h", false, "Show help")
	helpFlagLong := flag.Bool("help", false, "Show help")
	unlockOnly := flag.Bool("unlock-only", false, "Only unlock the PDF")
	extractOnly := flag.Bool("extract-only", false, "Only extract images")

	flag.Parse()

	// Show help if requested
	if *helpFlag || *helpFlagLong {
		printHelp()
		return
	}

	// Get remaining arguments
	args := flag.Args()

	// Validate arguments
	if len(args) < 1 {
		fmt.Println("Error: No PDF file specified")
		fmt.Println("Use 'pixf -h' for usage information")
		os.Exit(1)
	}

	filename := args[0]
	format := "original"

	// Get format from second argument if present and not an unlock-only operation
	if len(args) > 1 && !*unlockOnly {
		format = strings.TrimPrefix(args[1], "--")
	}

	// Validate format
	supportedFormats := []string{"original", "png", "webp"}
	isValidFormat := false
	for _, f := range supportedFormats {
		if format == f {
			isValidFormat = true
			break
		}
	}
	if !isValidFormat && !*unlockOnly {
		fmt.Printf("Error: Unsupported format '%s'\n", format)
		fmt.Println("Supported formats: original, png, webp")
		fmt.Println("Use 'pixf -h' for usage information")
		os.Exit(1)
	}

	// Handle unlock-only mode
	if *unlockOnly {
		fmt.Println("Unlocking PDF...")
		conf := model.NewDefaultConfiguration()
		filenameUnlocked := "unlocked_" + filename
		err := api.DecryptFile(filename, filenameUnlocked, conf)
		if err != nil {
			fmt.Println("Error decrypting PDF:", err)
			os.Exit(1)
		}
		fmt.Println("PDF successfully unlocked and saved as", filenameUnlocked)
		return
	}

	// Handle extract-only mode (use original PDF without unlocking)
	if *extractOnly {
		fmt.Println("Extracting images from:", filename)
		nameOnly := strings.TrimSuffix(filename, ".pdf")
		imgDir := "images_" + nameOnly

		err := imageHandling.ExtractImagesFromFile(filename, imgDir, format)
		if err != nil {
			fmt.Println("Error extracting images:", err)
			os.Exit(1)
		}
		fmt.Println("Images extracted to:", imgDir)
		return
	}

	// Default mode: unlock then extract images
	fmt.Println("Loading PDF:", filename)

	// PDFCPU Unlocking
	conf := model.NewDefaultConfiguration()
	filenameUnlocked := "unlocked_" + filename
	err := api.DecryptFile(filename, filenameUnlocked, conf)
	if err != nil {
		fmt.Println("Error decrypting PDF:", err)
		os.Exit(1)
	}
	fmt.Println("PDF successfully unlocked and saved as", filenameUnlocked)

	// PDFCPU Image Extraction
	nameOnly := strings.TrimSuffix(filename, ".pdf")
	imgDir := "images_" + nameOnly

	fmt.Println("Extracting images in", format, "format...")
	err = imageHandling.ExtractImagesFromFile(filenameUnlocked, imgDir, format)
	if err != nil {
		fmt.Println("Error extracting images:", err)
		os.Exit(1)
	}

	fmt.Println("Images extracted to:", imgDir)
}
