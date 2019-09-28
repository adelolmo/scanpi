package pdf

import (
	"fmt"
	"github.com/jung-kurt/gofpdf"
	"io"
	"path"
)

type Document struct {
	file *gofpdf.Fpdf
}

func NewPdfFile() *Document {
	return &Document{
		file: gofpdf.New("P", "mm", "A4", ""),
	}
}

func (d *Document) AddImage(imagePath string) error {
	imageType := path.Ext(imagePath)[1:]
	if imageType != "jpeg" && imageType != "png" {
		return fmt.Errorf("image format '%s' not supported", imageType)
	}
	d.file.AddPage()
	options := gofpdf.ImageOptions{ImageType: "jpeg", ReadDpi: true, AllowNegativePosition: false}
	d.file.ImageOptions(imagePath, 0, 0, 210, 295, false, options, 0, "")
	return nil
}

func (d *Document) Generate(w io.Writer) error {
	return d.file.Output(w)
}
