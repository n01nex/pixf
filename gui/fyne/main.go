package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	imageHandling "pixf/internal/toolset"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

func main() {
	a := app.NewWithID("com.pixf.gui")

	w := a.NewWindow("Pixf - PDF Image Extractor")

	// Variables to store state
	var selectedFile string
	var extractOnly bool
	var selectedFormat string = "original"
	var noBg bool
	var statusLabel *widget.Label
	var processButton *widget.Button
	var noBgCheck *widget.Check
	var fileLabel *widget.Label

	// Create status label first
	statusLabel = widget.NewLabel("")

	// Create noBg checkbox (disabled by default)
	noBgCheck = widget.NewCheck("No Background (transparent)", func(b bool) {
		noBg = b
	})
	noBgCheck.Disable()

	// Create process button
	processButton = widget.NewButton("Process PDF", func() {
		if selectedFile == "" {
			dialog.ShowError(fmt.Errorf("please select a PDF file"), w)
			return
		}

		// Check if file exists
		if _, err := os.Stat(selectedFile); os.IsNotExist(err) {
			dialog.ShowError(fmt.Errorf("file does not exist: %s", selectedFile), w)
			return
		}

		// Build format string
		format := selectedFormat
		if noBg && (selectedFormat == "png" || selectedFormat == "webp") {
			format = format + "-nobg"
		}

		statusLabel.SetText("Processing...")
		processButton.Disable()

		// Run processing in a goroutine to not freeze UI
		go func() {
			err := processPDF(selectedFile, format, extractOnly)

			// Use fyne.Do to safely update UI from goroutine
			fyne.Do(func() {
				processButton.Enable()
				if err != nil {
					statusLabel.SetText(fmt.Sprintf("Error: %v", err))
					dialog.ShowError(err, w)
				} else {
					statusLabel.SetText("Processing complete!")
					dialog.ShowInformation("Success", "PDF processed successfully!", w)
				}
			})
		}()
	})

	// Create file label
	fileLabel = widget.NewLabel("No file selected")

	// Create file button
	fileButton := widget.NewButton("Select PDF File", func() {
		dialog.ShowFileOpen(func(uri fyne.URIReadCloser, err error) {
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
			if uri == nil {
				return
			}
			selectedFile = uri.URI().Path()
			fileLabel.SetText(selectedFile)
		}, w)
	})

	// Extract only checkbox
	extractOnlyCheck := widget.NewCheck("Extract Only (skip unlock)", func(b bool) {
		extractOnly = b
	})

	// Format selection (radio buttons)
	formatLabel := widget.NewLabel("Output Format:")
	formatOriginal := widget.NewRadioGroup([]string{"original", "png", "webp"}, func(s string) {
		selectedFormat = s
		// Update noBg checkbox enabled state
		if s == "png" || s == "webp" {
			noBgCheck.Enable()
		} else {
			noBgCheck.Disable()
			noBgCheck.Checked = false
			noBg = false
		}
	})
	formatOriginal.Selected = "original"
	formatOriginal.Horizontal = true

	// Layout
	content := fyne.NewContainerWithLayout(
		layout.NewVBoxLayout(),
		widget.NewLabel("Pixf - PDF Image Extractor"),
		widget.NewSeparator(),
		fileButton,
		fileLabel,
		widget.NewSeparator(),
		extractOnlyCheck,
		widget.NewSeparator(),
		formatLabel,
		formatOriginal,
		widget.NewSeparator(),
		noBgCheck,
		widget.NewSeparator(),
		processButton,
		statusLabel,
	)

	w.SetContent(content)
	w.Resize(fyne.NewSize(500, 450))
	w.ShowAndRun()
}

func processPDF(filename string, format string, extractOnly bool) error {
	// Get just the filename without the full path
	basename := filepath.Base(filename)
	nameOnly := strings.TrimSuffix(basename, ".pdf")

	if extractOnly {
		// Extract only mode - skip unlocking
		fmt.Println("Extracting images from:", filename)
		imgDir := "images_" + nameOnly

		err := imageHandling.ExtractImagesFromFile(filename, imgDir, format)
		if err != nil {
			return fmt.Errorf("error extracting images: %w", err)
		}
		fmt.Println("Images extracted to:", imgDir)
		return nil
	}

	// Default mode: unlock then extract
	fmt.Println("Loading PDF:", filename)

	// PDFCPU Unlocking
	conf := model.NewDefaultConfiguration()
	filenameUnlocked := "unlocked_" + nameOnly + ".pdf"
	err := api.DecryptFile(filename, filenameUnlocked, conf)
	if err != nil {
		return fmt.Errorf("error decrypting PDF: %w", err)
	}
	fmt.Println("PDF successfully unlocked and saved as", filenameUnlocked)

	// PDFCPU Image Extraction
	imgDir := "images_" + nameOnly

	fmt.Println("Extracting images in", format, "format...")
	err = imageHandling.ExtractImagesFromFile(filenameUnlocked, imgDir, format)
	if err != nil {
		return fmt.Errorf("error extracting images: %w", err)
	}

	fmt.Println("Images extracted to:", imgDir)
	return nil
}
