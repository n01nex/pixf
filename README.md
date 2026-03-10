# pixf

**PDF Image Extract** - A simple yet powerful PDF toolkit for unlocking PDFs and extracting images, written in Go.

## Overview

pixf is a command-line tool built in Go that provides two primary functionalities:
- **Unlock PDFs**: Remove password protection from PDF files
- **Extract Images**: Extract all images from PDFs in your preferred format

## Motivation

While many PDF tools exist, none offered me a quick, command-line way to extract only unique images in the format I most commonly use. 
This tool fills that gap by eliminating duplicate images and allowing to output files in a format in an efficient way, taking advantage of Go parallel processing.

## Features

- 🔓 **Unlock PDFs** - Remove "Honor Mode" lock protection from PDF files
- 🖼️ **Extract Images** - Extract all images from PDF documents
- 📁 **Multiple Formats** - Extract as original format, PNG, or WebP
- 🚀 **Concurrent Processing** - Fast image extraction with parallel processing
- 🔄 **Deduplication** - Automatically removes duplicate images
- 🎨 **Background Removal** - Optional transparent backgrounds for PNG/WebP
- 🖥️ **Simple CLI** - Easy-to-use command-line interface

## Quick Start

### From Source

```bash
# Clone the repository
go install https://github.com/n01nex/pixf

# Optional: Add to PATH
mv pixf /usr/local/bin/
```

## Usage

```bash
pixf [OPTIONS] <pdf-file> [format]
```

### Arguments

| Argument | Description |
|----------|-------------|
| `pdf-file` | Path to the PDF file to process (required) |
| `format` | Image output format (optional, default: `original`) |

### Options

| Flag | Description |
|------|-------------|
| `-h, --help` | Show help message |
| `--unlock-only` | Only unlock the PDF, do not extract images |
| `--extract-only` | Only extract images, do not unlock the PDF first |
| `--nobg` | Remove background and create transparent PNG/WebP |

### Format Options

| Format | Description |
|--------|-------------|
| `original` | Extract images using PDF's native format (default) |
| `png` | Extract as PNG with transparency support |
| `png-nobg` | Extract as PNG with background removed (transparent) |
| `webp` | Extract as WebP with transparency support |
| `webp-nobg` | Extract as WebP with background removed (transparent) |

## Examples

### Basic Usage

```bash
# Unlock PDF and extract images (original format)
pixf document.pdf

# Extract images as PNG
pixf document.pdf png

# Extract images as WebP
pixf document.pdf webp
```

### Background Removal (Transparent)

```bash
# Extract as PNG with transparent background (using flag)
pixf document.pdf png --nobg

# Extract as WebP with transparent background (using flag)
pixf document.pdf webp --nobg

# Extract as PNG with transparent background (using format)
pixf document.pdf png-nobg

# Extract as WebP with transparent background (using format)
pixf document.pdf webp-nobg

# Short form (defaults to png)
pixf document.pdf --nobg
```

### Unlock Only Mode

```bash
# Only unlock the PDF (no image extraction)
pixf --unlock-only document.pdf
```

### Extract Only Mode

```bash
# Only extract images without unlocking
pixf --extract-only document.pdf
```

### Show Help

```bash
# Show help message
pixf -h
pixf --help
```

## How Background Removal Works

The `--nobg` flag attempts to create transparent backgrounds using two methods:

1. **Mask Detection**: If the PDF contains alpha channel masks, pixf detects and merges them with the base images
2. **Corner Color Detection**: If no masks are found, pixf samples the corner pixels to detect the background color and makes similar colors transparent

## Output

- Unlocked PDFs are saved as `unlocked_<original-filename>`
- Extracted images are saved in `images_<pdf-name>/` directory
- Duplicate images are automatically detected and skipped

## Dependencies

- [pdfcpu](https://github.com/pdfcpu/pdfcpu) - PDF processing library
- [chai2010/webp](https://github.com/chai2010/webp) - WebP encoding support

## Future Features

- [ ] Image upscale
- [ ] Image compression
- [ ] Image cropping
- [x] Background removal

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
