package fs

import (
	"io/ioutil"
	"log"
	"os"
	"path"
	"sort"
)

func ImageFilesOnDirectory(dir string) []os.FileInfo {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		log.Fatal(err)
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].Name() > files[j].Name()
	})
	i := 0
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		ext := path.Ext(file.Name())
		if ext != ".tiff" && ext != ".png" && ext != ".jpeg" && ext != ".pnm" && ext != ".pdf" {
			continue
		}
		files[i] = file
		i++

	}
	files = files[:i]
	return files
}
