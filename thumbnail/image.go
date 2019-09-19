package thumbnail

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/disintegration/imaging"
	"golang.org/x/image/tiff"
	"image/jpeg"
	"io/ioutil"
	"os"
)

func Preview(originalImage string) (*bytes.Buffer, error) {
	previewPath := originalImage + ".thumbnail"
	if _, err := os.Stat(previewPath); os.IsNotExist(err) {
		fmt.Println("Generate preview")
		//imagePath := path.Join(appConfiguration.OutputDirectory, jobName, scan)
		file, err := ioutil.ReadFile(originalImage)
		if err != nil {
			return nil, errors.New(fmt.Sprintf("Cannot read image on %s. Error: %s\n", originalImage, err))
		}
		srcImage, err := tiff.Decode(bytes.NewReader(file))
		if err != nil {
			return nil, errors.New(fmt.Sprintf("Cannot decode image on %s. Error: %s\n", originalImage, err))
		}
		dst := imaging.Resize(srcImage, 0, 500, imaging.Lanczos)
		err = imaging.Save(dst, previewPath+".jpeg")
		if err != nil {
			return nil, errors.New(fmt.Sprintf("Cannot save preview on %s. Error: %s\n", previewPath+".jpeg", err))
		}
		if err = os.Rename(previewPath+".jpeg", previewPath); err != nil {
			return nil, errors.New(fmt.Sprintf("Cannot rename thumbnail on %s. Error: %s\n", previewPath+".jpeg", err))
		}
	}

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
