package scanimage

import (
	"fmt"
	"io/ioutil"
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
		return Tiff
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

func (s *Scan) Start(path string) {
	go func() {
		fmt.Println("Scanning process. Start")
		// su -s /bin/sh - saned
		command := exec.Command("/usr/bin/scanimage",
			fmt.Sprintf("--mode=%s", s.Mode.String()),
			fmt.Sprintf("--resolution=%d", s.Resolution),
			fmt.Sprintf("--format=%s", s.Format.String()))
		fmt.Println(command.Args)
		out, err := command.Output()
		if err != nil {
			fmt.Println(fmt.Sprintf("Error executing scanimage command. Output: %s\n", out), err)
		}

		err = ioutil.WriteFile(path, out, 0644)
		if err != nil {
			fmt.Println(fmt.Sprintf("Cannot write image file on %s. Error: %s\n", path, err))
		}
		fmt.Println("Scanning process. End")
	}()
}
