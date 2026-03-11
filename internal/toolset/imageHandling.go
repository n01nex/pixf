package imageHandling

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"

	"github.com/chai2010/webp"
	"github.com/pdfcpu/pdfcpu/pkg/api"
)

// Buffer pool for encoding to reduce allocations
var bufferPool = sync.Pool{
	New: func() interface{} { return new(bytes.Buffer) },
}

func getBuffer() *bytes.Buffer { return bufferPool.Get().(*bytes.Buffer) }
func putBuffer(buf *bytes.Buffer) {
	buf.Reset()
	bufferPool.Put(buf)
}

// ImageEncoder interface
type ImageEncoder interface {
	Encode(w io.Writer, img *image.RGBA) error
	Extension() string
}

// Encoders
type PNGEncoder struct{}

func (PNGEncoder) Encode(w io.Writer, img *image.RGBA) error {
	encoder := png.Encoder{CompressionLevel: png.NoCompression}
	return encoder.Encode(w, img)
}
func (PNGEncoder) Extension() string { return ".png" }

type WebPEncoder struct{}

func (WebPEncoder) Encode(w io.Writer, img *image.RGBA) error {
	return webp.Encode(w, img, &webp.Options{Lossless: true, Quality: 100})
}
func (WebPEncoder) Extension() string { return ".webp" }

// Encoder registry
var encoderRegistry = map[string]ImageEncoder{
	"png":       PNGEncoder{},
	"png-nobg":  PNGEncoder{},
	"webp":      WebPEncoder{},
	"webp-nobg": WebPEncoder{},
}

// GetEncoder returns encoder for given format
func GetEncoder(format string) (ImageEncoder, error) {
	format = strings.ToLower(format)
	if enc, ok := encoderRegistry[format]; ok {
		return enc, nil
	}
	return nil, fmt.Errorf("unsupported format: %s", format)
}

// toRGBA converts any image to RGBA
func toRGBA(img image.Image) *image.RGBA {
	if img == nil {
		return nil
	}
	if rgba, ok := img.(*image.RGBA); ok {
		return rgba
	}
	b := img.Bounds()
	rgba := image.NewRGBA(b)
	draw.Draw(rgba, b, img, b.Min, draw.Over)
	return rgba
}

// isGrayscale checks if an image is grayscale (potential alpha mask)
func isGrayscale(img *image.RGBA) bool {
	if img == nil {
		return false
	}
	bounds := img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			// Check if R == G == B (grayscale)
			if r != g || g != b {
				return false
			}
		}
	}
	return true
}

// LoadedImage holds image data for processing
type LoadedImage struct {
	OrigName string      // Original filename for "original" format
	Img      *image.RGBA // Decoded RGBA (for conversion)
	RawData  []byte      // Original bytes (for "original" format)
	FileHash string
	IsMask   bool
	Width    int
	Height   int
}

// ExtractImagesFromFile extracts images from a PDF
func ExtractImagesFromFile(filename string, imgDir string, format string) error {
	if err := os.Mkdir(imgDir, 0755); err != nil && !os.IsExist(err) {
		return err
	}

	// Extract to temp directory
	tempDir, err := os.MkdirTemp("", "pdfimg")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	if err := api.ExtractImagesFile(filename, tempDir, nil, nil); err != nil {
		return fmt.Errorf("extract images: %w", err)
	}

	// Load all images
	images, err := loadImages(tempDir)
	if err != nil {
		return err
	}

	if len(images) == 0 {
		return nil
	}

	// Check if this is a nobg format
	isNobg := strings.HasSuffix(format, "-nobg")

	// For nobg format, try to merge masks
	if isNobg {
		images = mergeMasks(images)
	}

	// Deduplicate
	images = deduplicate(images)

	// Process based on format
	format = strings.ToLower(format)
	if format == "original" || format == "" {
		return saveOriginal(images, imgDir)
	}

	encoder, err := GetEncoder(format)
	if err != nil {
		return err
	}

	return saveConverted(images, imgDir, encoder, isNobg)
}

// mergeMasks attempts to detect and merge mask pairs
func mergeMasks(images []LoadedImage) []LoadedImage {
	if len(images) < 2 {
		return images
	}

	// Group images by dimensions
	dimMap := make(map[string][]int)
	for i, img := range images {
		if img.Img == nil {
			continue
		}
		key := fmt.Sprintf("%dx%d", img.Width, img.Height)
		dimMap[key] = append(dimMap[key], i)
	}

	used := make(map[int]bool)
	var result []LoadedImage

	for i, img := range images {
		if used[i] || img.Img == nil {
			continue
		}

		key := fmt.Sprintf("%dx%d", img.Width, img.Height)
		candidates := dimMap[key]

		for _, j := range candidates {
			if i == j || used[j] {
				continue
			}

			candidate := images[j]
			if candidate.Img == nil {
				continue
			}

			// If we find a grayscale image, try to merge it as alpha
			if isGrayscale(candidate.Img) && candidate.Width == img.Width && candidate.Height == img.Height {
				merged := applyAlphaMask(img.Img, candidate.Img)
				used[i] = true
				used[j] = true
				result = append(result, LoadedImage{
					OrigName: img.OrigName,
					Img:      merged,
					RawData:  nil,
					FileHash: "",
					IsMask:   false,
					Width:    img.Width,
					Height:   img.Height,
				})
				break
			}
		}

		if !used[i] {
			result = append(result, img)
		}
	}

	// If masks were found, they're already applied with transparency
	// If no masks were found, the original images are kept as-is
	// Background removal is handled in saveConverted instead
	return result
}

// applyAlphaMask applies a grayscale image as alpha channel
// Note: Assumes white in mask = transparent, black = opaque
func applyAlphaMask(base *image.RGBA, mask *image.RGBA) *image.RGBA {
	if base == nil || mask == nil {
		return base
	}
	b := base.Bounds()
	result := image.NewRGBA(b)

	draw.Draw(result, b, base, b.Min, draw.Src)

	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, _, _, _ := mask.At(x, y).RGBA()
			alpha := uint8(r >> 8)
			alpha = 255 - alpha // Invert: white=transparent

			old := result.At(x, y)
			r, g, b, _ := old.RGBA()
			result.SetRGBA(x, y, color.RGBA{
				R: uint8(r >> 8),
				G: uint8(g >> 8),
				B: uint8(b >> 8),
				A: alpha,
			})
		}
	}

	return result
}

// removeBackground removes background by detecting edge color and using threshold-based comparison
func removeBackground(img *image.RGBA) *image.RGBA {
	if img == nil {
		return nil
	}
	bounds := img.Bounds()
	if bounds.Empty() {
		return img
	}

	// Detect background color from all edges (not just corners)
	// Also try common background colors like white and black as fallback
	bgColors, dominantRatio := detectBackgroundFromEdges(img)

	// Only process if dominant edge color ratio is very high (95%)
	if dominantRatio < 0.95 {
		return img
	}

	// Check if dominant color is near-white OR pure black
	// We want to remove white backgrounds and pure black backgrounds
	// But NOT near-black (dark gray) which might be content
	isNearBlackOrWhite := false
	for _, bg := range bgColors {
		// Check if near-white (R,G,B all > 200)
		if bg.R > 200 && bg.G > 200 && bg.B > 200 {
			isNearBlackOrWhite = true
			break
		}
		// Check if PURE black only (R,G,B all == 0) - not near-black
		if bg.R == 0 && bg.G == 0 && bg.B == 0 {
			isNearBlackOrWhite = true
			break
		}
	}

	if !isNearBlackOrWhite {
		return img // Not black or white background, skip processing
	}

	// Add both white and black as fallbacks
	bgColors = append(bgColors, color.RGBA{255, 255, 255, 255})
	bgColors = append(bgColors, color.RGBA{0, 0, 0, 255})

	if len(bgColors) == 0 {
		return img
	}

	// Use simple pixel-by-pixel threshold-based removal
	return simpleBackgroundRemoval(img, bgColors)
}

// detectBackgroundFromEdges samples the edges of the image to find the background color
// Returns the dominant colors and the ratio of the most common color to total edge pixels
func detectBackgroundFromEdges(img *image.RGBA) ([]color.RGBA, float64) {
	if img == nil {
		return nil, 0
	}
	bounds := img.Bounds()

	// Collect color samples from all edges
	var edgeColors []color.RGBA

	// Sample top and bottom edges
	for x := bounds.Min.X; x < bounds.Max.X; x++ {
		// Top edge - include all pixels (even transparent ones)
		r, g, b, a := img.At(x, bounds.Min.Y).RGBA()
		edgeColors = append(edgeColors, color.RGBA{
			R: uint8(r >> 8),
			G: uint8(g >> 8),
			B: uint8(b >> 8),
			A: uint8(a >> 8),
		})
		// Bottom edge (skip if same as top edge for single-row images)
		if bounds.Max.Y-1 != bounds.Min.Y {
			r, g, b, a := img.At(x, bounds.Max.Y-1).RGBA()
			edgeColors = append(edgeColors, color.RGBA{
				R: uint8(r >> 8),
				G: uint8(g >> 8),
				B: uint8(b >> 8),
				A: uint8(a >> 8),
			})
		}
	}

	// Sample left and right edges (excluding corners to avoid double-counting)
	for y := bounds.Min.Y + 1; y < bounds.Max.Y-1; y++ {
		// Left edge (excluding corners)
		r, g, b, a := img.At(bounds.Min.X, y).RGBA()
		edgeColors = append(edgeColors, color.RGBA{
			R: uint8(r >> 8),
			G: uint8(g >> 8),
			B: uint8(b >> 8),
			A: uint8(a >> 8),
		})
		// Right edge (excluding corners)
		if bounds.Max.X-1 != bounds.Min.X {
			r, g, b, a := img.At(bounds.Max.X-1, y).RGBA()
			edgeColors = append(edgeColors, color.RGBA{
				R: uint8(r >> 8),
				G: uint8(g >> 8),
				B: uint8(b >> 8),
				A: uint8(a >> 8),
			})
		}
	}

	if len(edgeColors) == 0 {
		return nil, 0
	}

	// Find the most common color among edge samples
	colors, maxCount := findDominantColors(edgeColors, 3)
	dominantRatio := float64(maxCount) / float64(len(edgeColors))

	return colors, dominantRatio
}

// findDominantColors finds the most common colors in a slice
// Returns the dominant colors and the count of the most common color
func findDominantColors(colors []color.RGBA, maxColors int) ([]color.RGBA, int) {
	if len(colors) == 0 {
		return nil, 0
	}

	// Group similar colors together
	colorGroups := make(map[string]int)
	for _, c := range colors {
		// Quantize colors to group similar ones (divide by 8 to reduce precision)
		key := fmt.Sprintf("%d,%d,%d", c.R/8, c.G/8, c.B/8)
		colorGroups[key]++
	}

	// Sort by frequency
	type colorCount struct {
		color color.RGBA
		count int
	}
	var sorted []colorCount
	for k, v := range colorGroups {
		var r, g, b int
		fmt.Sscanf(k, "%d,%d,%d", &r, &g, &b)
		sorted = append(sorted, colorCount{
			color: color.RGBA{uint8(r * 8), uint8(g * 8), uint8(b * 8), 255},
			count: v,
		})
	}

	// Sort by count descending
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].count > sorted[j].count
	})

	// Get max count
	maxCount := 0
	if len(sorted) > 0 {
		maxCount = sorted[0].count
	}

	// Return top colors
	result := make([]color.RGBA, 0, maxColors)
	for i := 0; i < len(sorted) && i < maxColors; i++ {
		result = append(result, sorted[i].color)
	}

	return result, maxCount
}

// simpleBackgroundRemoval is a fallback method using threshold-based removal
func simpleBackgroundRemoval(img *image.RGBA, bgColors []color.RGBA) *image.RGBA {
	if img == nil {
		return img
	}
	if len(bgColors) == 0 {
		return img
	}

	bounds := img.Bounds()
	if bounds.Empty() {
		return img
	}

	result := image.NewRGBA(bounds)

	scaledThreshold := uint32(3000)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()

			// If already transparent, preserve
			if a < 32768 {
				result.SetRGBA(x, y, color.RGBA{
					R: uint8(r >> 8),
					G: uint8(g >> 8),
					B: uint8(b >> 8),
					A: uint8(a >> 8),
				})
				continue
			}

			// Check if matches any background color
			isBackground := false
			for _, bg := range bgColors {
				bgR := uint32(bg.R) << 8
				bgG := uint32(bg.G) << 8
				bgB := uint32(bg.B) << 8

				if diff(r, bgR) < scaledThreshold && diff(g, bgG) < scaledThreshold && diff(b, bgB) < scaledThreshold {
					isBackground = true
					break
				}
			}

			if isBackground {
				result.SetRGBA(x, y, color.RGBA{0, 0, 0, 0})
			} else {
				result.SetRGBA(x, y, color.RGBA{
					R: uint8(r >> 8),
					G: uint8(g >> 8),
					B: uint8(b >> 8),
					A: uint8(a >> 8),
				})
			}
		}
	}

	return result
}

func diff(a, b uint32) uint32 {
	if a > b {
		return a - b
	}
	return b - a
}

// loadImages reads and decodes all image files
func loadImages(dir string) ([]LoadedImage, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read dir: %w", err)
	}

	var images []LoadedImage
	for _, f := range files {
		if !isImageFile(f.Name()) {
			continue
		}

		path := filepath.Join(dir, f.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", f.Name(), err)
		}

		img, _, err := image.Decode(bytes.NewReader(data))
		if err != nil {
			continue
		}

		rgba := toRGBA(img)
		if rgba == nil {
			continue
		}

		images = append(images, LoadedImage{
			OrigName: f.Name(),
			Img:      rgba,
			RawData:  data,
			FileHash: hashBytes(data),
			IsMask:   false,
			Width:    rgba.Bounds().Dx(),
			Height:   rgba.Bounds().Dy(),
		})
	}
	return images, nil
}

// deduplicate removes duplicate images by hash
func deduplicate(images []LoadedImage) []LoadedImage {
	seen := make(map[string]bool)
	var unique []LoadedImage

	for _, img := range images {
		if img.FileHash != "" && seen[img.FileHash] {
			continue
		}
		if img.FileHash != "" {
			seen[img.FileHash] = true
		}
		unique = append(unique, img)
	}

	if len(unique) < len(images) {
		fmt.Printf("skipped %d duplicate(s)\n", len(images)-len(unique))
	}
	return unique
}

// saveOriginal copies raw files
func saveOriginal(images []LoadedImage, imgDir string) error {
	for i, img := range images {
		ext := strings.ToLower(filepath.Ext(img.OrigName))
		if ext == "" {
			ext = ".png"
		}
		path := filepath.Join(imgDir, fmt.Sprintf("image_%04d%s", i+1, ext))
		if err := os.WriteFile(path, img.RawData, 0644); err != nil {
			return fmt.Errorf("write %s: %w", path, err)
		}
	}
	return nil
}

// saveConverted encodes images concurrently
func saveConverted(images []LoadedImage, imgDir string, encoder ImageEncoder, isNobg bool) error {
	numWorkers := runtime.NumCPU()

	type task struct {
		index int
		img   *image.RGBA
	}

	tasks := make(chan task, len(images))
	results := make(chan error, len(images))

	var wg sync.WaitGroup

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for t := range tasks {
				processedImg := t.img
				if isNobg && processedImg != nil {
					processedImg = removeBackground(processedImg)
				}
				results <- encodeImage(processedImg, encoder, imgDir, t.index)
			}
		}()
	}

	go func() {
		for i, img := range images {
			tasks <- task{index: i, img: img.Img}
		}
		close(tasks)
	}()

	go func() {
		wg.Wait()
		close(results)
	}()

	var firstErr error
	for err := range results {
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// encodeImage encodes a single image to disk
func encodeImage(img *image.RGBA, encoder ImageEncoder, imgDir string, index int) error {
	if img == nil {
		return nil
	}
	buf := getBuffer()
	defer putBuffer(buf)

	if err := encoder.Encode(buf, img); err != nil {
		return fmt.Errorf("encode: %w", err)
	}

	outPath := filepath.Join(imgDir, fmt.Sprintf("image_%04d%s", index+1, encoder.Extension()))
	return os.WriteFile(outPath, buf.Bytes(), 0644)
}

// isImageFile checks if filename has image extension
func isImageFile(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	return ext == ".png" || ext == ".jpg" || ext == ".jpeg" ||
		ext == ".gif" || ext == ".bmp" || ext == ".tiff" || ext == ".webp"
}

// hashBytes computes SHA-256 of data
func hashBytes(data []byte) string {
	h := sha256.New()
	h.Write(data)
	return fmt.Sprintf("%x", h.Sum(nil))
}
