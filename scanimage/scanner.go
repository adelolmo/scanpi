package scanimage

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os/exec"
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

func (s *Scan) Start(path string) error {
	// su -s /bin/sh - saned
	out, err := exec.Command("/usr/bin/scanimage",
		fmt.Sprintf("--mode=%s", s.Mode.String()),
		"--resolution=300",
		fmt.Sprintf("--format=%s", s.Format.String())).Output()
	if err != nil {
		return errors.New(err.Error() + ". " + string(out))
	}

	err = ioutil.WriteFile(path, out, 0644)
	if err != nil {
		return err
	}
	return nil
}
