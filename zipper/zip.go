package zipper

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
)

type ZipWriter struct {
	w *zip.Writer
}

func NewZipper(writer io.Writer) ZipWriter {
	return ZipWriter{
		w: zip.NewWriter(writer),
	}
}

func (z *ZipWriter) AddFile(path string, filename string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open %s: %s", path, err)
	}

	wr, err := z.w.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create entry for %s in zip file: %s", path, err)
	}

	if _, err := io.Copy(wr, file); err != nil {
		return fmt.Errorf("failed to write %s to zip: %s", path, err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("failed to close file %s to zip: %s", file.Name(), err)
	}

	return nil
}

func (z *ZipWriter) Close() error {
	return z.w.Close()
}
