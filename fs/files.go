package fs

import (
	"fmt"
	"github.com/adelolmo/sane-web-client/debug"
	"io/ioutil"
	"os"
	"path"
	"sort"
)

func ImageFilesOnDirectory(dir string) []os.FileInfo {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		debug.Error(fmt.Sprintf("unable to get images from directory '%s'", dir))
		return []os.FileInfo{}
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
