package main

import (
	"encoding/json"
	"fmt"
	"github.com/adelolmo/sane-web-client/fs"
	"github.com/adelolmo/sane-web-client/scanimage"
	"github.com/adelolmo/sane-web-client/thumbnail"
	"github.com/gobuffalo/packr"
	"github.com/gorilla/mux"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
)

type settings struct {
	Navigation string `json:"-"`
	Mode       string `json:"mode"`
	Format     string `json:"format"`
	Resolution string `json:"resolution"`
	Updated    bool   `json:"-"`
}

type pageJobs struct {
	Navigation   string
	JobName      string
	PreviousJobs []string
	Scans        []string
	JobStarted   bool
}

type pageScanner struct {
	Navigation string
	JobName    string
	Scans      []string
	JobStarted bool
}

var indexTemplate *template.Template
var jobTemplate *template.Template
var jobsTemplate *template.Template
var settingsTemplate *template.Template

type configuration struct {
	OutputDirectory string
	WorkDirectory   string
}

var appConfiguration configuration

func init() {
	box := packr.NewBox("./templates")
	indexFile, err := box.FindString("index.html")
	if err != nil {
		log.Fatalln(err)
	}
	headerFile, err := box.FindString("header.html")
	if err != nil {
		log.Fatalln(err)
	}
	indexTemplate = template.Must(template.Must(template.New("index").Parse(headerFile)).Parse(indexFile))

	jobFile, err := box.FindString("job.html")
	if err != nil {
		log.Fatalln(err)
	}
	jobTemplate = template.Must(template.Must(template.New("job").Parse(headerFile)).Parse(jobFile))

	jobsFile, err := box.FindString("jobs.html")
	if err != nil {
		log.Fatalln(err)
	}
	jobsTemplate = template.Must(template.Must(template.New("jobs").Parse(headerFile)).Parse(jobsFile))

	settingsFile, err := box.FindString("settings.html")
	if err != nil {
		log.Fatalln(err)
	}
	settingsTemplate = template.Must(template.Must(template.New("settings").Parse(headerFile)).Parse(settingsFile))

}

func main() {
	port := os.Getenv("port")
	if port == "" {
		port = "8000"
	}
	outputDirectory := os.Getenv("outputDir")
	workDirectory := os.Getenv("workDir")
	appConfiguration = configuration{
		OutputDirectory: outputDirectory,
		WorkDirectory:   workDirectory,
	}
	fmt.Println(fmt.Sprintf("port: %s, outputDir: %s, workDir: %s", port, outputDirectory, workDirectory))

	settingsFile := path.Join(appConfiguration.WorkDirectory, "settings.json")
	if _, err := os.Stat(settingsFile); os.IsNotExist(err) {
		settings := &settings{
			Mode:       scanimage.Color.String(),
			Format:     scanimage.Tiff.String(),
			Resolution: "300",
		}
		settingsJson, _ := json.Marshal(settings)
		if err := ioutil.WriteFile(path.Join(appConfiguration.WorkDirectory, "settings.json"), settingsJson, 0644); err != nil {
			log.Fatalln(err)
		}
	}

	box := packr.NewBox("assets/")

	router := mux.NewRouter()
	router.PathPrefix("/assets/").Handler(http.StripPrefix("/assets/", http.FileServer(box)))
	router.HandleFunc("/", homePage).Methods("GET")
	router.HandleFunc("/settings", showSettingsPage).Methods("GET")
	router.HandleFunc("/settings", updateSettingsPage).Methods("POST")
	router.HandleFunc("/jobs", showJobsPage).Methods("GET")
	router.HandleFunc("/job", resumeJobPage).Methods("GET")
	router.HandleFunc("/job", createJobHandler).Methods("POST")
	router.HandleFunc("/deleteJob", deleteJobHandler).Methods("POST")
	router.HandleFunc("/scan", scanHandler).Methods("POST")
	router.HandleFunc("/deleteScan", deleteScanHandler).Methods("POST")
	router.HandleFunc("/download", downloadFileHandler).Methods("GET")
	router.HandleFunc("/preview", previewHandler).Methods("GET")
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), router))
}

func homePage(w http.ResponseWriter, r *http.Request) {
	var previousJobs []string
	for _, dir := range fs.JobDirectories(appConfiguration.OutputDirectory) {
		previousJobs = append(previousJobs, dir.Name())
	}

	type index struct {
		Navigation string
	}
	err := indexTemplate.Execute(w, &index{Navigation: "home"})
	if err != nil {
		fmt.Println(fmt.Sprintf("Cannot execute index template. Error: %s", err.Error()))
	}
}

func showSettingsPage(w http.ResponseWriter, r *http.Request) {
	settings := readSettings()
	err := settingsTemplate.Execute(w, settings)
	if err != nil {
		panic(err)
	}
}

func updateSettingsPage(w http.ResponseWriter, r *http.Request) {
	mode := r.FormValue("mode")
	format := r.FormValue("format")
	resolution := r.FormValue("resolution")
	settings := &settings{
		Navigation: "settings",
		Mode:       mode,
		Format:     format,
		Resolution: resolution,
		Updated:    true,
	}
	settingsJson, _ := json.Marshal(settings)
	if err := ioutil.WriteFile(path.Join(appConfiguration.WorkDirectory, "settings.json"), settingsJson, 0644); err != nil {
		log.Fatalln(err)
	}

	if err := settingsTemplate.Execute(w, settings); err != nil {
		log.Fatalln(err)
	}
}

func showJobsPage(w http.ResponseWriter, r *http.Request) {
	var previousJobs []string
	for _, dir := range fs.JobDirectories(appConfiguration.OutputDirectory) {
		previousJobs = append(previousJobs, dir.Name())
	}

	index := &pageJobs{
		Navigation:   "jobs",
		PreviousJobs: previousJobs,
	}
	err := jobsTemplate.Execute(w, index)
	if err != nil {
		log.Fatalln(err)
	}
}

func resumeJobPage(w http.ResponseWriter, r *http.Request) {
	jobName := r.FormValue("jobName")

	if err := os.MkdirAll(path.Join(appConfiguration.OutputDirectory, jobName), os.ModePerm); err != nil {
		log.Fatalln(err)
	}

	var scans []string
	directory := fs.ImageFilesOnDirectory(path.Join(appConfiguration.OutputDirectory, jobName))
	for _, file := range directory {
		scans = append(scans, file.Name())
	}

	scanner := &pageJobs{
		Navigation: "jobs",
		JobName:    jobName,
		Scans:      scans,
	}
	if err := jobTemplate.Execute(w, scanner); err != nil {
		log.Fatalln(err)
	}
}

func createJobHandler(w http.ResponseWriter, r *http.Request) {
	jobName := r.FormValue("jobName")
	if err := os.MkdirAll(path.Join(appConfiguration.OutputDirectory, jobName), os.ModePerm); err != nil {
		log.Fatalln(err)
	}

	var scans []string
	for _, file := range fs.ImageFilesOnDirectory(path.Join(appConfiguration.OutputDirectory, jobName)) {
		scans = append(scans, file.Name())
	}

	scanner := &pageScanner{
		Navigation: "jobs",
		Scans:      scans,
		JobName:    jobName,
	}
	if err := jobTemplate.Execute(w, scanner); err != nil {
		log.Fatalln(err)
	}
}

func deleteJobHandler(w http.ResponseWriter, r *http.Request) {
	jobName := r.FormValue("jobName")

	jobPath := path.Join(appConfiguration.OutputDirectory, jobName)
	if err := os.RemoveAll(jobPath); err != nil {
		log.Println(fmt.Sprintf("unable to delete job directory %s.", jobPath), err)
	}

	var previousJobs []string
	for _, dir := range fs.JobDirectories(appConfiguration.OutputDirectory) {
		previousJobs = append(previousJobs, dir.Name())
	}

	index := &pageJobs{
		Navigation:   "jobs",
		PreviousJobs: previousJobs,
	}
	err := jobsTemplate.Execute(w, index)
	if err != nil {
		log.Fatalln(err)
	}
}

func scanHandler(w http.ResponseWriter, r *http.Request) {
	jobName := r.FormValue("jobName")
	previousScans := fs.ImageFilesOnDirectory(path.Join(appConfiguration.OutputDirectory, jobName))

	fileExtension := readSettings().Format
	scanName := fmt.Sprintf("1.%s", fileExtension)
	if len(previousScans) > 0 {
		lastScanName := previousScans[0].Name()
		lastScanNumber, err := strconv.Atoi(strings.Split(lastScanName, ".")[0])
		if err != nil {
			println(err)
		}
		scanName = fmt.Sprintf("%d.%s", lastScanNumber+1, fileExtension)
	}

	settings := readSettings()
	resolution, _ := strconv.Atoi(settings.Resolution)
	scanJob := scanimage.NewScanJob(
		scanimage.ToMode(settings.Mode),
		scanimage.ToFormat(settings.Format),
		resolution)
	scanJob.Start(path.Join(appConfiguration.OutputDirectory, jobName, scanName))

	var scans []string
	for _, file := range previousScans {
		scans = append(scans, file.Name())
	}

	scanner := &pageJobs{
		Navigation: "jobs",
		JobName:    jobName,
		Scans:      scans,
		JobStarted: true,
	}
	if err := jobTemplate.Execute(w, scanner); err != nil {
		log.Fatalln(err)
	}
}

func deleteScanHandler(w http.ResponseWriter, r *http.Request) {
	jobName := r.FormValue("jobName")
	scan := r.FormValue("scan")
	imagePath := path.Join(appConfiguration.OutputDirectory, jobName, scan)
	fmt.Printf("delete image %s\n", imagePath)

	if err := os.Remove(imagePath); err != nil {
		log.Println(fmt.Sprintf("unable to delete image file %s.", imagePath), err)
	}
	if err := thumbnail.DeletePreview(imagePath); err != nil {
		log.Println(err.Error())
	}

	var scans []string
	directory := fs.ImageFilesOnDirectory(path.Join(appConfiguration.OutputDirectory, jobName))
	for _, file := range directory {
		scans = append(scans, file.Name())
	}

	scanner := &pageJobs{
		Navigation: "jobs",
		JobName:    jobName,
		Scans:      scans,
	}
	if err := jobTemplate.Execute(w, scanner); err != nil {
		log.Fatalln(err)
	}
}

func downloadFileHandler(w http.ResponseWriter, r *http.Request) {
	jobName := r.FormValue("jobName")
	scan := r.FormValue("scan")

	imagePath := path.Join(appConfiguration.OutputDirectory, jobName, scan)
	file, err := ioutil.ReadFile(imagePath)
	if err != nil {
		log.Println(fmt.Sprintf("unable to read image file %s.", imagePath), err)
	}

	contentType := fmt.Sprintf("image/%s", path.Ext(imagePath)[1:])
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", strconv.Itoa(len(file)))
	w.Header().Set("content-disposition",
		fmt.Sprintf("attachment; filename=\"%s-%s\"", jobName, scan))
	if _, err := w.Write(file); err != nil {
		log.Println(fmt.Sprintf("unable to stream image %s.", imagePath), err)
	}
}

func previewHandler(w http.ResponseWriter, r *http.Request) {
	jobName := r.FormValue("jobName")
	scan := r.FormValue("scan")

	w.Header().Set("Content-Type", "image/jpeg")

	imagePath := path.Join(appConfiguration.OutputDirectory, jobName, scan)
	buffer, err := thumbnail.Preview(imagePath)
	if err != nil {
		log.Println(fmt.Sprintf("unable to get thumbnail %s. Error: %s", imagePath, err.Error()))
		box := packr.NewBox("./assets")
		b, err := box.Find("not_available.jpeg")
		if err != nil {
			log.Fatalln("Cannot read asset not_available.jpeg")
		}
		if _, err := w.Write(b); err != nil {
			log.Println(fmt.Sprintf("unable to stream image %s.", imagePath), err)
		}
		w.Header().Set("Content-Length", strconv.Itoa(len(b)))
		return
	}

	w.Header().Set("Content-Length", strconv.Itoa(len(buffer.Bytes())))
	if _, err := w.Write(buffer.Bytes()); err != nil {
		fmt.Printf("failed to served preview: %v\n", err)
		w.WriteHeader(http.StatusNotFound)
		return
	}
}

func readSettings() *settings {
	settingsFile := path.Join(appConfiguration.WorkDirectory, "settings.json")
	file, err := ioutil.ReadFile(settingsFile)
	if err != nil {
		log.Fatalln(err)
	}
	settings := &settings{
		Navigation: "settings",
	}
	if err = json.Unmarshal(file, settings); err != nil {
		log.Fatalln(err)
	}
	return settings
}
