package graphic

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/adelolmo/scanpi/fsutils"
	"github.com/adelolmo/scanpi/logger"
	"github.com/disintegration/imaging"
	"golang.org/x/image/tiff"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"path"
	"time"
)

type Thumbnail struct {
	filter        imaging.ResampleFilter
	baseDirectory string
}

func NewThumbnail(thumbnailName, baseDirectory string) *Thumbnail {
	return &Thumbnail{
		filter:        toThumbnailFilter(thumbnailName),
		baseDirectory: baseDirectory,
	}
}

func (t Thumbnail) Preview(originalImage string) (*bytes.Buffer, error) {
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

func (t Thumbnail) GenerateThumbnail(imageDetails ImageDetails) error {
	start := time.Now()

	previewPath := imageDetails.ImagePath() + ".thumbnail"

	logger.Info("(%s) Generating preview...", imageDetails.Filename())

	logger.Info("(%s) read file", imageDetails.Filename())
	file, err := os.ReadFile(imageDetails.ImagePath())
	if err != nil {
		return errors.New(fmt.Sprintf("Cannot read image on %s. Error: %s", imageDetails.Filename(), err))
	}
	logger.Info("(%s) done", imageDetails.Filename())

	logger.Info("(%s) decode image", imageDetails.Filename())
	srcImage, err := decodeImage(bytes.NewReader(file), imageDetails.ImagePath())
	if err != nil {
		return errors.New(fmt.Sprintf("Cannot decode image on %s. Error: %s", imageDetails.Filename(), err))
	}
	logger.Info("(%s) done", imageDetails.Filename())

	logger.Info("(%s) resize image", imageDetails.Filename())
	dst := imaging.Resize(srcImage, 0, 250, t.filter)
	logger.Info("(%s) done", imageDetails.Filename())

	logger.Info("(%s) save image", imageDetails.Filename())
	err = imaging.Save(dst, previewPath+".jpeg")
	if err != nil {
		return errors.New(fmt.Sprintf("Cannot save preview on %s. Error: %s\n", previewPath+".jpeg", err))
	}
	logger.Info("(%s) done", imageDetails.Filename())

	logger.Info("(%s) rename", imageDetails.Filename())
	if err = os.Rename(previewPath+".jpeg", previewPath); err != nil {
		return errors.New(fmt.Sprintf("Cannot rename Thumbnail on %s. Error: %s", previewPath+".jpeg", err))
	}

	logger.Info("(%s) creating symlink", imageDetails.Filename())
	err = os.Symlink(imageDetails.Filename()+".thumbnail", imageDetails.LinkPath()+".thumbnail")
	if err != nil {
		logger.Error(fmt.Sprintf("Cannot create symlink to image file on '%s'. Error: %s", imageDetails.Filename(), err))
	}
	logger.Info("(%s) done", imageDetails.Filename())

	logger.Info("(%s) Generation took %fs", imageDetails.Filename(), time.Now().Sub(start).Seconds())

	return nil
}

func (t Thumbnail) DeletePreview(originalImage string) error {
	previewPath := originalImage + ".thumbnail"
	return fsutils.DeleteFileAndLink(previewPath)
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

func toThumbnailFilter(filter string) imaging.ResampleFilter {
	switch filter {
	case "NearestNeighbor":
		return imaging.NearestNeighbor
	case "Box":
		return imaging.Box
	case "Linear":
		return imaging.Linear
	case "Hermite":
		return imaging.Hermite
	case "MitchellNetravali":
		return imaging.MitchellNetravali
	case "CatmullRom":
		return imaging.CatmullRom
	case "BSpline":
		return imaging.BSpline
	case "Gaussian":
		return imaging.Gaussian
	case "Bartlett":
		return imaging.Bartlett
	case "Lanczos":
		return imaging.Lanczos
	case "Hann":
		return imaging.Hann
	case "Hamming":
		return imaging.Hamming
	case "Blackman":
		return imaging.Blackman
	case "Welch":
		return imaging.Welch
	case "Cosine":
		return imaging.Cosine
	default:
		logger.Info("using default filter NearestNeighbor instead of unknown %s", filter)
		return imaging.NearestNeighbor
	}
}
