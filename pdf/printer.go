package pdf

import (
	"github.com/jung-kurt/gofpdf"
	"io"
)

type File struct {
	document *gofpdf.Fpdf
}

func NewPdfFile() *File {
	return &File{
		document: gofpdf.New("P", "mm", "A4", ""),
	}
}

func (p *File) AddImage(imagePath string) {
	p.document.AddPage()
	options := gofpdf.ImageOptions{ImageType: "jpeg", ReadDpi: true, AllowNegativePosition: false}
	p.document.ImageOptions(imagePath, 0, 0, 210, 295, false, options, 0, "")
}

func (p *File) Generate(w io.Writer) error {
	return p.document.Output(w)
}
