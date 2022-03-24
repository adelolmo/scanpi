package fsutils

import (
	"io/ioutil"
	"log"
	"os"
	"sort"
)

func JobDirectories(dir string) []os.FileInfo {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		log.Fatal(err)
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].ModTime().After(files[j].ModTime())
	})
	i := 0
	for _, file := range files {
		if file.IsDir() {
			files[i] = file
			i++
		}
	}
	files = files[:i]
	return files
}
