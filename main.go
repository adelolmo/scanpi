package main

import (
	"fmt"
	"github.com/gobuffalo/packr"
	"github.com/gorilla/mux"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"sort"
	"strconv"
	"strings"
)

type index struct {
	Title        string
	JobName      string
	PreviousJobs []string
}

type scanner struct {
	JobName string
	Scans   []string
}

var indexTemplate *template.Template
var jobTemplate *template.Template
var outputDirectory string

func init() {
	box := packr.NewBox("./templates")
	indexFile, err := box.FindString("index.html")
	if err != nil {
		log.Fatalln(err)
	}
	indexTemplate = template.Must(template.New("index.html").Parse(indexFile))

	jobFile, err := box.FindString("job.html")
	if err != nil {
		log.Fatalln(err)
	}
	jobTemplate = template.Must(template.New("job.html").Parse(jobFile))
}

func main() {
	port := os.Getenv("port")
	if port == "" {
		port = "8000"
	}
	outputDirectory = os.Getenv("outputDir")
	fmt.Println(fmt.Sprintf("port: %s, outputDir: %s", port, outputDirectory))

	box := packr.NewBox("./assets")

	router := mux.NewRouter().StrictSlash(true)
	router.Handle("/assets", http.FileServer(box))
	router.HandleFunc("/", homePage).Methods("GET")
	router.HandleFunc("/job", resumeJobPage).Methods("GET")
	router.HandleFunc("/job", createJobHandler).Methods("POST")
	router.HandleFunc("/scan", scanHandler).Methods("POST")
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), router))

}

func homePage(w http.ResponseWriter, r *http.Request) {
	var previousJobs []string
	for _, dir := range jobDirectories() {
		previousJobs = append(previousJobs, dir.Name())
	}

	index := &index{
		PreviousJobs: previousJobs,
	}
	err := indexTemplate.Execute(w, index)
	if err != nil {
		panic(err)
	}
}

func resumeJobPage(w http.ResponseWriter, r *http.Request) {
	if err := os.MkdirAll(path.Join(outputDirectory, r.FormValue("jobName")), os.ModePerm); err != nil {
		log.Fatalln(err)
	}

	var scans []string
	directory := filesOnDirectory(path.Join(outputDirectory, r.FormValue("jobName")))
	for _, file := range directory {
		scans = append(scans, file.Name())
		println(file.Name())
	}

	scanner := &scanner{
		Scans:   scans,
		JobName: r.FormValue("jobName"),
	}
	if err := jobTemplate.Execute(w, scanner); err != nil {
		log.Fatalln(err)
	}
}

func createJobHandler(w http.ResponseWriter, r *http.Request) {
	jobName := r.FormValue("jobName")
	if err := os.MkdirAll(path.Join(outputDirectory, jobName), os.ModePerm); err != nil {
		log.Fatalln(err)
	}

	var scans []string
	for _, file := range filesOnDirectory(path.Join(outputDirectory, jobName)) {
		scans = append(scans, file.Name())
	}

	scanner := &scanner{
		Scans:   scans,
		JobName: jobName,
	}
	if err := jobTemplate.Execute(w, scanner); err != nil {
		log.Fatalln(err)
	}
}

func scanHandler(w http.ResponseWriter, r *http.Request) {
	jobName := r.FormValue("jobName")
	previousScans := filesOnDirectory(path.Join(outputDirectory, jobName))

	scanName := "1.tiff"
	/*for _, scanFilename := range previousScans {
	}*/
	if len(previousScans) > 0 {
		lastScanName := previousScans[len(previousScans)-1].Name()
		lastScanNumber, err := strconv.Atoi(strings.Split(lastScanName, ".")[0])
		if err != nil {
			println(err)
		}
		scanName = fmt.Sprintf("%d.tiff", lastScanNumber+1)
	}

	err := scan(path.Join(outputDirectory, jobName, scanName))
	if err != nil {
		log.Fatalln(err)
	}

	var scans []string
	for _, file := range previousScans {
		scans = append(scans, file.Name())
	}

	scanner := &scanner{
		Scans:   scans,
		JobName: jobName,
	}
	if err := jobTemplate.Execute(w, scanner); err != nil {
		log.Fatalln(err)
	}
}

func scan(path string) error {
	// su -s /bin/sh - saned
	out, err := exec.Command("/usr/bin/scanimage",
		"--mode=Color", "--resolution=300", "--format=tiff").Output()
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(path, out, 0644)
	if err != nil {
		return err
	}
	return nil
}

func jobDirectories() []os.FileInfo {
	files, err := ioutil.ReadDir(outputDirectory)
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

func filesOnDirectory(dir string) []os.FileInfo {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		log.Fatal(err)
	}
	sort.Slice(files, func(i, j int) bool {
		//return sort.Strings(files[i].Name(), files[j].Name())
		return files[i].Name() > files[j].Name()
	})
	//sort.Strings()
	i := 0
	for _, file := range files {
		if !file.IsDir() {
			files[i] = file
			i++
		}
	}
	files = files[:i]
	return files
}
