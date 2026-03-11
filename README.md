# pixf

**PDF Image Extract** - A simple yet powerful PDF toolkit for unlocking PDFs and extracting images, written in Go.

## Overview

pixf is a PDF toolkit written in Go that provides two primary functionalities:
- **Unlock PDFs**: Remove password protection from PDF files
- **Extract Images**: Extract all images from PDFs in your preferred format

pixf offers two interfaces: a **Command-Line Interface (CLI)** for power users and a **Graphical User Interface (GUI)** built with Fyne for a more accessible experience.

## Screenshots

<!-- Add your screenshot here -->
<!-- Recommended size: 800x600 or similar aspect ratio -->
![pixf GUI Screenshot](docs/screenshot.png)

*Above: pixf GUI (Fyne version)*

---

## CLI Version

The CLI version is designed for power users who prefer working in the terminal. It offers full control over all features through command-line arguments.

### Installation

```bash
# Clone the repository
git clone https://github.com/n01nex/pixf.git

# Build the CLI version
cd pixf
go build -o pixf main.go

# Optional: Add to PATH
sudo mv pixf /usr/local/bin/
```

### CLI Usage

```bash
pixf [OPTIONS] <pdf-file> [format]
```

#### Arguments

| Argument | Description |
|----------|-------------|
| `pdf-file` | Path to the PDF file to process (required) |
| `format` | Image output format (optional, default: `original`) |

#### Options

| Flag | Description |
|------|-------------|
| `-h, --help` | Show help message |
| `--unlock-only` | Only unlock the PDF, do not extract images |
| `--extract-only` | Only extract images, do not unlock the PDF first |
| `--nobg` | Remove background and create transparent PNG/WebP |

#### Format Options

| Format | Description |
|--------|-------------|
| `original` | Extract images using PDF's native format (default) |
| `png` | Extract as PNG with transparency support |
| `png-nobg` | Extract as PNG with background removed (transparent) |
| `webp` | Extract as WebP with transparency support |
| `webp-nobg` | Extract as WebP with background removed (transparent) |

### CLI Examples

```bash
# Unlock PDF and extract images (original format)
pixf document.pdf

# Extract images as PNG
pixf document.pdf png

# Extract images as WebP
pixf document.pdf webp

# Extract as PNG with transparent background (using flag)
pixf document.pdf png --nobg

# Extract as WebP with transparent background (using flag)
pixf document.pdf webp --nobg

# Short form (defaults to png)
pixf document.pdf --nobg

# Only unlock the PDF (no image extraction)
pixf --unlock-only document.pdf

# Only extract images without unlocking
pixf --extract-only document.pdf

# Show help message
pixf -h
pixf --help
```

### How the CLI Works

1. **Input Validation**: The CLI parses arguments and validates the input file exists
2. **PDF Unlocking**: Uses pdfcpu to decrypt/unlock the PDF (unless `--extract-only` is specified)
3. **Image Extraction**: Extracts all images from the PDF to a temporary directory
4. **Format Conversion**: If a format is specified (png/webp), converts the extracted images
5. **Background Removal**: If `--nobg` is specified, attempts to make backgrounds transparent
6. **Deduplication**: Detects and removes duplicate images using SHA256 hashing
7. **Output**: Saves unlocked PDF and extracted images to the working directory

---

## GUI Version (Fyne)

The GUI version provides a user-friendly graphical interface built with [Fyne](https://fyne.io/), a cross-platform Go UI framework. It runs as a separate executable from `gui/fyne/`.

### Installation

```bash
# Clone the repository
git clone https://github.com/n01nex/pixf.git

# Build the GUI version
cd pixf/gui/fyne
go build -o pixf-gui
```

### GUI Features

The GUI provides a simple, intuitive interface with the following controls:

- **File Selection**: Click "Select PDF File" to open a file dialog
- **Output Format**: Choose between original, PNG, or WebP using radio buttons
- **Extract Only Mode**: Checkbox to skip PDF unlocking
- **No Background**: Checkbox to enable transparent background (PNG/WebP only)
- **Process Button**: Click to start processing the selected PDF

### How the GUI Works

1. **File Selection**: Opens a native file dialog to select a PDF file
2. **Configuration**: User selects output format and optional features via checkboxes
3. **Processing**: Clicking "Process PDF" runs the processing in a background goroutine to keep the UI responsive
4. **Status Feedback**: Shows real-time status updates and success/error messages
5. **Output**: Creates the same output as the CLI (unlocked PDF + images folder)

### GUI vs CLI Comparison

| Feature | CLI | GUI |
|---------|-----|-----|
| File Selection | Command-line argument | File dialog |
| Format Selection | Command argument | Radio buttons |
| No Background | `--nobg` flag | Checkbox |
| Unlock Only | `--unlock-only` flag | N/A (use Extract Only) |
| Extract Only | `--extract-only` flag | Checkbox |
| Progress Feedback | Terminal output | Status label |

---

## How Background Removal Works

The `--nobg` flag (CLI) or "No Background" checkbox (GUI) attempts to create transparent backgrounds using two methods:

1. **Mask Detection**: If the PDF contains alpha channel masks, pixf detects and merges them with the base images
2. **Corner Color Detection**: If no masks are found, pixf samples the edge pixels to detect the background color and makes similar colors transparent

The background removal only applies to PNG and WebP output formats.

---

## Output

- Unlocked PDFs are saved as `unlocked_<original-filename>.pdf`
- Extracted images are saved in `images_<pdf-name>/` directory
- Duplicate images are automatically detected and skipped

---

## Dependencies

- [pdfcpu](https://github.com/pdfcpu/pdfcpu) - PDF processing library
- [chai2010/webp](https://github.com/chai2010/webp) - WebP encoding support
- [fyne.io/fyne](https://fyne.io/) - GUI framework (GUI version only)

---

## Future Features

- [ ] Image upscale
- [ ] Image compression
- [ ] Image cropping
- [x] Background removal

---

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
