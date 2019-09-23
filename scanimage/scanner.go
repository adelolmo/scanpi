package scanimage

import (
	"fmt"
	"github.com/adelolmo/sane-web-client/thumbnail"
	"github.com/jung-kurt/gofpdf"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
)

type Mode int

const (
	Lineart Mode = iota
	Gray
	Color
)

func (m Mode) String() string {
	switch m {
	case Color:
		return "Color"
	case Gray:
		return "Gray"
	case Lineart:
		return "Lineart"
	default:
		return "Gray"
	}
}

func ToMode(mode string) Mode {
	formattedMode := strings.Title(mode)
	if formattedMode == "Color" {
		return Color
	}
	if formattedMode == "Gray" {
		return Gray
	}
	if formattedMode == "Lineart" {
		return Lineart
	} else {
		return Color
	}
}

type Format int

const (
	Tiff Format = iota
	Png
	Jpeg
	Pnm
	Pdf
)

func (f Format) String() string {
	switch f {
	case Tiff:
		return "tiff"
	case Jpeg:
		return "jpeg"
	case Png:
		return "png"
	case Pnm:
		return "pnm"
	case Pdf:
		return "pdf"
	default:
		return "tiff"
	}
}

func ToFormat(format string) Format {
	formattedMode := strings.Title(format)
	if formattedMode == "Tiff" {
		return Tiff
	}
	if formattedMode == "Jpeg" {
		return Jpeg
	}
	if formattedMode == "Png" {
		return Png
	}
	if formattedMode == "Pnm" {
		return Pnm
	}
	if formattedMode == "Pdf" {
		return Pdf
	} else {
		return Jpeg
	}
}

type Scan struct {
	Mode       Mode
	Format     Format
	Resolution int
}

func NewScanJob(mode Mode, format Format, resolution int) *Scan {
	return &Scan{Format: format,
		Mode:       mode,
		Resolution: resolution}
}

func (s *Scan) Start(imagePath string) {
	go func() {
		fmt.Println(fmt.Sprintf("Scanning process for %s. Start", imagePath))

		format := s.Format
		if s.Format == Pdf {
			format = Jpeg
		}

		// su -s /bin/sh - saned
		command := exec.Command("/usr/bin/scanimage",
			fmt.Sprintf("--mode=%s", s.Mode.String()),
			fmt.Sprintf("--resolution=%d", s.Resolution),
			fmt.Sprintf("--format=%s", format.String()))
		fmt.Println(command.Args)
		out, err := command.Output()
		if err != nil {
			fmt.Println(fmt.Sprintf("Error executing scanimage command. Output: %s\n", out), err)
			return
		}

		err = ioutil.WriteFile(imagePath, out, 0644)
		if err != nil {
			fmt.Println(fmt.Sprintf("Cannot write image file on %s. Error: %s\n", imagePath, err))
		}

		if s.Format == Pdf {
			jpegImage := strings.Replace(imagePath, Pdf.String(), Jpeg.String(), 1)
			if err = thumbnail.GenerateThumbnail(jpegImage); err != nil {
				fmt.Println(err.Error())
			}
			generatePdfFromJpeg(jpegImage)
			return
		}

		if err = thumbnail.GenerateThumbnail(imagePath); err != nil {
			fmt.Println(err.Error())
		}

		fmt.Println(fmt.Sprintf("Scanning process for %s. End", imagePath))
	}()
}

func generatePdfFromJpeg(imagePath string) {
	fmt.Println(fmt.Sprintf("Pdf generation for %s. Start", imagePath))

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	options := gofpdf.ImageOptions{ImageType: Jpeg.String(), ReadDpi: true, AllowNegativePosition: false}
	pdf.ImageOptions(imagePath, 0, 0, 0, 0, false, options, 0, "")
	pdfFile := strings.Replace(imagePath, Jpeg.String(), Pdf.String(), 1)
	if err := pdf.OutputFileAndClose(pdfFile); err != nil {
		fmt.Println(fmt.Sprintf("Cannot write pdf file on %s. Error: %s\n", pdfFile, err.Error()))
	}
	if err := os.Remove(imagePath); err != nil {
		fmt.Println(fmt.Sprintf("Cannot remove original image file on %s. Error: %s\n", imagePath, err.Error()))
	}

	fmt.Println(fmt.Sprintf("Pdf generation for %s. End", imagePath))
}
