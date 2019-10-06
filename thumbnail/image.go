package thumbnail

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/adelolmo/sane-web-client/debug"
	"github.com/disintegration/imaging"
	"golang.org/x/image/tiff"
	"image"
	"image/jpeg"
	"image/png"
	"io/ioutil"
	"os"
	"path"
	"time"
)

func Preview(originalImage string) (*bytes.Buffer, error) {
	previewPath := originalImage + ".thumbnail"
	preview, err := imaging.Open(previewPath)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("failed to open image: %v\n", err))

	}
	buf := new(bytes.Buffer)
	if err := jpeg.Encode(buf, preview, nil); err != nil {
		return nil, errors.New(fmt.Sprintf("failed to encode preview: %v\n", err))

	}
	return buf, nil
}

func GenerateThumbnail(baseDir, imageFilename string) error {
	start := time.Now()

	previewPath := path.Join(baseDir, imageFilename) + ".thumbnail"

	debug.Info(fmt.Sprintf("(%s) Generating preview...", imageFilename))

	debug.Info(fmt.Sprintf("(%s) read file", imageFilename))
	file, err := ioutil.ReadFile(path.Join(baseDir, imageFilename))
	if err != nil {
		return errors.New(fmt.Sprintf("Cannot read image on %s. Error: %s", imageFilename, err))
	}
	debug.Info(fmt.Sprintf("(%s) done", imageFilename))

	debug.Info(fmt.Sprintf("(%s) decode image", imageFilename))
	srcImage, err := decodeImage(bytes.NewReader(file), path.Join(baseDir, imageFilename))
	if err != nil {
		return errors.New(fmt.Sprintf("Cannot decode image on %s. Error: %s", imageFilename, err))
	}
	debug.Info(fmt.Sprintf("(%s) done", imageFilename))

	debug.Info(fmt.Sprintf("(%s) resize image", imageFilename))
	dst := imaging.Resize(srcImage, 0, 250, imaging.NearestNeighbor)
	debug.Info(fmt.Sprintf("(%s) done", imageFilename))

	debug.Info(fmt.Sprintf("(%s) save image", imageFilename))
	err = imaging.Save(dst, previewPath+".jpeg")
	if err != nil {
		return errors.New(fmt.Sprintf("Cannot save preview on %s. Error: %s\n", previewPath+".jpeg", err))
	}
	debug.Info(fmt.Sprintf("(%s) done", imageFilename))

	debug.Info(fmt.Sprintf("(%s) rename", imageFilename))
	if err = os.Rename(previewPath+".jpeg", previewPath); err != nil {
		return errors.New(fmt.Sprintf("Cannot rename thumbnail on %s. Error: %s", previewPath+".jpeg", err))
	}
	debug.Info(fmt.Sprintf("(%s) done", imageFilename))

	debug.Info(fmt.Sprintf("(%s) Generation took %fs", imageFilename, time.Now().Sub(start).Seconds()))

	return nil
}

func DeletePreview(originalImage string) error {
	previewPath := originalImage + ".thumbnail"

	if err := os.Remove(previewPath); err != nil {
		return errors.New(fmt.Sprintf("unable to delete image file %s. Error: %v", previewPath, err))
	}
	return nil
}

func decodeImage(r *bytes.Reader, originalImage string) (image.Image, error) {
	ext := path.Ext(originalImage)
	switch ext {
	case ".tiff":
		srcImage, err := tiff.Decode(r)
		if err != nil {
			return nil, err
		}
		return srcImage, nil
	case ".jpeg":
		srcImage, err := jpeg.Decode(r)
		if err != nil {
			return nil, err
		}
		return srcImage, nil
	case ".png":
		srcImage, err := png.Decode(r)
		if err != nil {
			return nil, err
		}
		return srcImage, nil
	}
	return nil, errors.New(fmt.Sprintf("image format not supported: %s\n", ext))
}
