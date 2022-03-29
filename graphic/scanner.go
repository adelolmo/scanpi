package graphic

import (
	"errors"
	"fmt"
	"github.com/adelolmo/scanpi/debug"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
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
	} else {
		return Jpeg
	}
}

type scan struct {
	mode       Mode
	format     Format
	resolution int
	thumbnail  *Thumbnail
}

type ImageDetails struct {
	Name          string
	LinkName      string
	Format        Format
	Directory     string
	BaseDirectory string
}

func (d ImageDetails) Filename() string {
	return d.Name + d.Format.Extension()
}

func (d ImageDetails) ImagePath() string {
	return filepath.Join(d.BaseDirectory, d.Directory, d.Filename())
}

func (d ImageDetails) LinkFilename() string {
	return d.LinkName + d.Format.Extension()
}

func (d ImageDetails) LinkPath() string {
	return filepath.Join(d.BaseDirectory, d.Directory, d.LinkFilename())
}

func NewScanJob(mode Mode, format Format, resolution int, thumbnail *Thumbnail) *scan {
	return &scan{format: format,
		mode:       mode,
		resolution: resolution,
		thumbnail:  thumbnail,
	}
}

func (s scan) StartScanning(imageDetails ImageDetails) {
	go func() {
		debug.Info(fmt.Sprintf("Scanning process for '%s' with symlink '%s'. Start",
			imageDetails.Filename(), imageDetails.LinkFilename()))

		// su -s /bin/sh - saned
		command := exec.Command("/usr/bin/scanimage",
			fmt.Sprintf("--mode=%s", s.mode.String()),
			fmt.Sprintf("--resolution=%d", s.resolution),
			fmt.Sprintf("--format=%s", s.format.String()))
		debug.Info(strings.Join(command.Args, " "))
		out, err := command.Output()
		if err != nil {
			debug.Error(fmt.Sprintf("Error executing scanimage command. Output: %s. Error:%v", out, err))
			return
		}

		err = ioutil.WriteFile(imageDetails.ImagePath(), out, 0644)
		if err != nil {
			debug.Error(fmt.Sprintf("Cannot write image file on '%s'. Error: %s", imageDetails.LinkFilename(), err))
		}

		err = os.Symlink(imageDetails.Filename(), imageDetails.LinkPath())
		if err != nil {
			debug.Error(fmt.Sprintf("Cannot create symlink to image file on '%s'. Error: %s", imageDetails.LinkPath(), err))
		}

		debug.Info(fmt.Sprintf("Scanning process for '%s'. End", imageDetails.LinkFilename()))

		// TODO extract and run after this async execution
		if err = s.thumbnail.GenerateThumbnail(imageDetails); err != nil {
			debug.Error(err.Error())
		}
	}()
}

func ScannerDevice() (string, error) {
	// scanimage -f "scanner number %i device %d is a %t, model %m, produced by %v"
	// scanimage -f "%m"
	command := exec.Command("/usr/bin/scanimage", "--formatted-device-list", "\"%m\"")
	debug.Info(strings.Join(command.Args, " "))
	out, err := command.Output()
	if err != nil {
		return "", errors.New(fmt.Sprintf("Error executing scanimage command. Output: %s. Error:%v", out, err))
	}
	if len(out) == 0 {
		return "", errors.New("no device available")
	}
	return string(out), nil
}
