package scanimage

import (
	"fmt"
	"github.com/adelolmo/sane-web-client/debug"
	"github.com/adelolmo/sane-web-client/thumbnail"
	"github.com/jung-kurt/gofpdf"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
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

func (f Format) Extension() string {
	return "." + f.String()
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

func (s *Scan) Start(baseDir string, imageFilename string) {
	go func() {
		debug.Info(fmt.Sprintf("Scanning process for %s. Start", imageFilename))

		format := s.Format
		outImageFilename := path.Join(baseDir, imageFilename)
		if s.Format == Pdf {
			debug.Info("Output format is pdf. Generate jpeg first.")
			format = Jpeg
			outImageFilename = path.Join(baseDir, "temp.jpeg")
			debug.Info(fmt.Sprintf("new imageFilename: %s.", outImageFilename))
		}

		// su -s /bin/sh - saned
		command := exec.Command("/usr/bin/scanimage",
			fmt.Sprintf("--mode=%s", s.Mode.String()),
			fmt.Sprintf("--resolution=%d", s.Resolution),
			fmt.Sprintf("--format=%s", format.String()))
		debug.Info(strings.Join(command.Args, " "))
		out, err := command.Output()
		if err != nil {
			debug.Error(fmt.Sprintf("Error executing scanimage command. Output: %s. Error:%v", out, err))
			return
		}

		err = ioutil.WriteFile(outImageFilename, out, 0644)
		if err != nil {
			debug.Error(fmt.Sprintf("Cannot write image file on %s. Error: %s", imageFilename, err))
		}

		if s.Format == Pdf {
			generatePdfFromJpeg(outImageFilename, path.Join(baseDir, imageFilename))
			debug.Info(fmt.Sprintf("Scanning process for %s. End", imageFilename))

			if err = thumbnail.GenerateThumbnail(outImageFilename, path.Join(baseDir, imageFilename)); err != nil {
				debug.Error(err.Error())
			}
			if err := os.Remove(outImageFilename); err != nil {
				debug.Error(fmt.Sprintf("Cannot remove original image file on %s. Error: %s", outImageFilename, err.Error()))
			}
			return
		}
		debug.Info(fmt.Sprintf("Scanning process for %s. End", imageFilename))

		if err = thumbnail.GenerateThumbnail(outImageFilename, path.Join(baseDir, imageFilename)); err != nil {
			debug.Error(err.Error())
		}
	}()
}

func Device() string {
	// scanimage -f "scanner number %i device %d is a %t, model %m, produced by %v"
	// scanimage -f "%m"
	command := exec.Command("/usr/bin/scanimage", "--formatted-device-list", "\"%m\"")
	debug.Info(strings.Join(command.Args, " "))
	out, err := command.Output()
	if err != nil {
		debug.Error(fmt.Sprintf("Error executing scanimage command. Output: %s. Error:%v", out, err))
		return ""
	}
	return string(out)
}

func generatePdfFromJpeg(srcImagePath string, outPdfPath string) {
	debug.Info(fmt.Sprintf("Pdf generation from image %s to pdf%s. Start", srcImagePath, outPdfPath))

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	options := gofpdf.ImageOptions{ImageType: Jpeg.String(), ReadDpi: true, AllowNegativePosition: false}
	pdf.ImageOptions(srcImagePath, 0, 0, 210, 295, false, options, 0, "")
	if err := pdf.OutputFileAndClose(outPdfPath); err != nil {
		debug.Error(fmt.Sprintf("Cannot write pdf file on %s. Error: %s", outPdfPath, err.Error()))
	}
	debug.Info(fmt.Sprintf("Pdf generation for %s. End", srcImagePath))
}
