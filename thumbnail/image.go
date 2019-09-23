package thumbnail

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/disintegration/imaging"
	"golang.org/x/image/tiff"
	"image"
	"image/jpeg"
	"image/png"
	"io/ioutil"
	"os"
	"path"
)

func Preview(originalImage string) (*bytes.Buffer, error) {
	previewPath := originalImage + ".thumbnail"
	/*	if _, err := os.Stat(previewPath); os.IsNotExist(err) {
		buffer, err := GenerateThumbnail(originalImage)
		if err != nil {
			return buffer, err
		}
	}*/

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

func GenerateThumbnail(originalImage string) error {
	previewPath := originalImage + ".thumbnail"

	fmt.Println(fmt.Sprintf("Generating preview for %s. Start", originalImage))
	fmt.Println("create empty thumbnail")
	_, err := os.Create(previewPath)
	if err != nil {
		return errors.New(fmt.Sprintf("Cannot create empty thumbnail for %s. Error: %v\n", originalImage, err))
	}
	fmt.Println("done")
	fmt.Println("read file")
	file, err := ioutil.ReadFile(originalImage)
	if err != nil {
		return errors.New(fmt.Sprintf("Cannot read image on %s. Error: %s\n", originalImage, err))
	}
	fmt.Println("done")
	fmt.Println("decode image")
	srcImage, err := decodeImage(file, originalImage)
	if err != nil {
		return errors.New(fmt.Sprintf("Cannot decode image on %s. Error: %s\n", originalImage, err))
	}
	fmt.Println("done")
	fmt.Println("resize image")
	dst := imaging.Resize(srcImage, 0, 300, imaging.Box)
	fmt.Println("done")
	fmt.Println("save image")
	err = imaging.Save(dst, previewPath+".jpeg")
	if err != nil {
		return errors.New(fmt.Sprintf("Cannot save preview on %s. Error: %s\n", previewPath+".jpeg", err))
	}
	fmt.Println("done")
	fmt.Println("rename")
	if err = os.Rename(previewPath+".jpeg", previewPath); err != nil {
		return errors.New(fmt.Sprintf("Cannot rename thumbnail on %s. Error: %s\n", previewPath+".jpeg", err))
	}
	fmt.Println("done")
	fmt.Println(fmt.Sprintf("Generating preview for %s. End", originalImage))
	return nil
}

func DeletePreview(originalImage string) error {
	previewPath := originalImage + ".thumbnail"

	if err := os.Remove(previewPath); err != nil {
		return errors.New(fmt.Sprintf("unable to delete image file %s. Error: %v", previewPath, err))
	}
	return nil
}

func decodeImage(file []byte, originalImage string) (image.Image, error) {
	ext := path.Ext(originalImage)
	switch ext {
	case ".tiff":
		srcImage, err := tiff.Decode(bytes.NewReader(file))
		if err != nil {
			return nil, err
		}
		return srcImage, nil
	case ".jpeg":
		srcImage, err := jpeg.Decode(bytes.NewReader(file))
		if err != nil {
			return nil, err
		}
		return srcImage, nil
	case ".png":
		srcImage, err := png.Decode(bytes.NewReader(file))
		if err != nil {
			return nil, err
		}
		return srcImage, nil
	}
	return nil, errors.New(fmt.Sprintf("Image format not supported: %s\n", ext))
}
