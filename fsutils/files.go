package fsutils

import (
	"errors"
	"fmt"
	"github.com/adelolmo/scanpi/logger"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sort"
	"time"
)

type FileMetaData struct {
	Filename string
	LinkName string
}

func ImageFilesOnDirectory(dir string) ([]FileMetaData, error) {
	metaData := make([]FileMetaData, 0)
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		logger.Error(fmt.Sprintf("unable to get images from directory '%s'", dir))
		return []FileMetaData{}, err
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].ModTime().Before(files[j].ModTime())
	})
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		if file.Mode()&os.ModeSymlink == 0 {
			continue
		}
		readlink, err := os.Readlink(filepath.Join(dir, file.Name()))
		if err != nil {
			fmt.Println("error: " + err.Error())
			continue
		}

		ext := path.Ext(readlink)
		if ext != ".tiff" && ext != ".png" && ext != ".jpeg" && ext != ".pnm" && ext != ".pdf" {
			continue
		}
		metaData = append(metaData, FileMetaData{
			Filename: readlink,
			LinkName: file.Name(),
		})

	}
	return metaData, nil
}

func GenerateDateFilename() string {
	return time.Now().Format("20060102150405")
}

func DeleteFileAndLink(filePath string) error {
	readlink, err := os.Readlink(filePath)
	if err != nil {
		return errors.New(fmt.Sprintf("unable to resolve symlink %s. error: %v", filePath, err))
	}

	// delete file
	if err := os.Remove(path.Join(path.Dir(filePath), readlink)); err != nil {
		return errors.New(fmt.Sprintf("unable to delete file %s. Error: %v", readlink, err))
	}
	
	// delete symlink
	if err := os.Remove(filePath); err != nil {
		return errors.New(fmt.Sprintf("unable to delete symlink %s. error: %v", filePath, err))
	}

	return nil
}
