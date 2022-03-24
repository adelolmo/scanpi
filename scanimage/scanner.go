package scanimage

import (
	"errors"
	"fmt"
	"github.com/adelolmo/scanpi/debug"
	"github.com/adelolmo/scanpi/thumbnail"
	"io/ioutil"
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
	thumbnail  *thumbnail.Thumbnail
}

func NewScanJob(mode Mode, format Format, resolution int, thumbnail *thumbnail.Thumbnail) *scan {
	return &scan{format: format,
		mode:       mode,
		resolution: resolution,
		thumbnail:  thumbnail,
	}
}

func (s *scan) Start(baseDir string, imageFilename string) {
	go func() {
		debug.Info(fmt.Sprintf("Scanning process for '%s'. Start", imageFilename))

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

		err = ioutil.WriteFile(path.Join(baseDir, imageFilename), out, 0644)
		if err != nil {
			debug.Error(fmt.Sprintf("Cannot write image file on '%s'. Error: %s", imageFilename, err))
		}

		debug.Info(fmt.Sprintf("Scanning process for '%s'. End", imageFilename))

		// TODO extract and run after this async execution
		if err = s.thumbnail.GenerateThumbnail(imageFilename); err != nil {
			debug.Error(err.Error())
		}
	}()
}

func Device() (string, error) {
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
