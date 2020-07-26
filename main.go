package main

import (
	"encoding/json"
	"fmt"
	"github.com/adelolmo/sane-web-client/debug"
	"github.com/adelolmo/sane-web-client/fs"
	"github.com/adelolmo/sane-web-client/pdf"
	"github.com/adelolmo/sane-web-client/scanimage"
	"github.com/adelolmo/sane-web-client/thumbnail"
	"github.com/adelolmo/sane-web-client/zipper"
	"github.com/gobuffalo/packr/v2"
	"github.com/gorilla/mux"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
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

type configuration struct {
	OutputDirectory string
	WorkDirectory   string
}

var indexTemplate *template.Template
var jobTemplate *template.Template
var jobsTemplate *template.Template
var settingsTemplate *template.Template

var appConfiguration configuration

func init() {
	box := packr.New("templates", "./templates")
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
	fmt.Println(fmt.Sprintf("port: %s, outputDir: %s, workDir: %s, debug: %v",
		port, outputDirectory, workDirectory, debug.Enabled()))

	settingsFile := path.Join(appConfiguration.WorkDirectory, "settings.json")
	if _, err := os.Stat(settingsFile); os.IsNotExist(err) {
		settings := &settings{
			Mode:       scanimage.Color.String(),
			Format:     scanimage.Jpeg.String(),
			Resolution: "200",
		}
		settingsJson, _ := json.Marshal(settings)
		if err := ioutil.WriteFile(path.Join(appConfiguration.WorkDirectory, "settings.json"), settingsJson, 0644); err != nil {
			log.Fatalln(err)
		}
	}

	box := packr.New("assets", "assets/")

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
	router.HandleFunc("/downloadall", downloadAllHandler).Methods("GET")
	router.HandleFunc("/preview", previewHandler).Methods("GET")

	router.HandleFunc("/scanner", scannerHandler).Methods("GET")

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
	if err := indexTemplate.Execute(w, &index{Navigation: "home"}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func showSettingsPage(w http.ResponseWriter, r *http.Request) {
	settings := readSettings()
	if err := settingsTemplate.Execute(w, settings); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := settingsTemplate.Execute(w, settings); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func resumeJobPage(w http.ResponseWriter, r *http.Request) {
	encodedJobName := r.FormValue("jobName")
	jobName, err := url.QueryUnescape(encodedJobName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func createJobHandler(w http.ResponseWriter, r *http.Request) {
	jobName := r.FormValue("jobName")
	if len(jobName) == 0 {
		http.Error(w, "jobName cannot be empty", http.StatusBadRequest)
		return
	}
	if err := os.MkdirAll(path.Join(appConfiguration.OutputDirectory, jobName), os.ModePerm); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func deleteJobHandler(w http.ResponseWriter, r *http.Request) {
	jobName := r.FormValue("jobName")

	jobPath := path.Join(appConfiguration.OutputDirectory, jobName)
	if err := os.RemoveAll(jobPath); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func scanHandler(w http.ResponseWriter, r *http.Request) {
	jobName := r.FormValue("jobName")
	previousScans := fs.ImageFilesOnDirectory(path.Join(appConfiguration.OutputDirectory, jobName))

	fileExtension := readSettings().Format
	scanName := fmt.Sprintf("1.%s", fileExtension)
	if len(previousScans) > 0 {
		lastScanName := previousScans[len(previousScans)-1].Name()
		lastScanNumber, err := strconv.Atoi(strings.Split(lastScanName, ".")[0])
		if err != nil {
			fmt.Println(err)
		}
		scanName = fmt.Sprintf("%d.%s", lastScanNumber+1, fileExtension)
	}

	settings := readSettings()
	resolution, _ := strconv.Atoi(settings.Resolution)
	scanJob := scanimage.NewScanJob(
		scanimage.ToMode(settings.Mode),
		scanimage.ToFormat(settings.Format),
		resolution)
	scanJob.Start(appConfiguration.OutputDirectory, path.Join(jobName, scanName))

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
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
func deleteScanHandler(w http.ResponseWriter, r *http.Request) {
	jobName := r.FormValue("jobName")
	scan := r.FormValue("scan")
	imagePath := path.Join(appConfiguration.OutputDirectory, jobName, scan)

	debug.Info(fmt.Sprintf("delete image %s\n", imagePath))

	if err := os.Remove(imagePath); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := thumbnail.DeletePreview(imagePath); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
func downloadFileHandler(w http.ResponseWriter, r *http.Request) {
	debug.Info("downloadFileHandler")
	encodedJobName := r.FormValue("jobName")
	jobName, err := url.QueryUnescape(encodedJobName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	scan := r.FormValue("scan")

	imagePath := path.Join(appConfiguration.OutputDirectory, jobName, scan)
	debug.Info(fmt.Sprintf("imagePath: %s", imagePath))
	file, err := ioutil.ReadFile(imagePath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", contentType(imagePath))
	w.Header().Set("Content-Length", strconv.Itoa(len(file)))
	w.Header().Set("content-disposition",
		fmt.Sprintf("attachment; filename=\"%s-%s\"", jobName, scan))
	if _, err := w.Write(file); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func downloadAllHandler(w http.ResponseWriter, r *http.Request) {
	debug.Info("downloadAllHandler")
	envelope := r.FormValue("envelope")
	encodedJobName := r.FormValue("jobName")
	jobName, err := url.QueryUnescape(encodedJobName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var scans []string
	directory := fs.ImageFilesOnDirectory(path.Join(appConfiguration.OutputDirectory, jobName))
	for _, file := range directory {
		scans = append(scans, file.Name())
	}

	switch envelope {
	case "zip":
		w.Header().Set("content-disposition", fmt.Sprintf("attachment; filename=\"%s.zip\"", jobName))
		zip := zipper.NewZipper(w)
		for _, filename := range scans {
			if err := zip.AddFile(path.Join(appConfiguration.OutputDirectory, jobName, filename), filename); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		if err := zip.Close(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	case "pdf":
		w.Header().Set("content-disposition", fmt.Sprintf("attachment; filename=\"%s.pdf\"", jobName))
		pdfFile := pdf.NewPdfFile()
		for _, filename := range scans {
			if err := pdfFile.AddImage(path.Join(appConfiguration.OutputDirectory, jobName, filename)); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		if err := pdfFile.Generate(w); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func previewHandler(w http.ResponseWriter, r *http.Request) {
	jobName := r.FormValue("jobName")
	scan := r.FormValue("scan")

	w.Header().Set("Content-Type", "image/jpeg")

	imagePath := path.Join(appConfiguration.OutputDirectory, jobName, scan)
	buffer, err := thumbnail.Preview(imagePath)
	if err != nil {
		box := packr.New("assets", "./assets")
		b, err := box.Find("not_available.jpeg")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if _, err := w.Write(b); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Length", strconv.Itoa(len(b)))
		return
	}

	w.Header().Set("Content-Length", strconv.Itoa(len(buffer.Bytes())))
	if _, err := w.Write(buffer.Bytes()); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func scannerHandler(w http.ResponseWriter, r *http.Request) {
	type scanner struct {
		Name   string `json:"name"`
		Status string `json:"status"`
	}

	jsonBody := scanner{
		Name:   "Unknown",
		Status: "Not available",
	}
	deviceName, err := scanimage.Device()
	if err == nil {
		jsonBody = scanner{
			Name:   deviceName,
			Status: "Available",
		}
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(jsonBody); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func readSettings() *settings {
	settingsFile := path.Join(appConfiguration.WorkDirectory, "settings.json")
	file, err := ioutil.ReadFile(settingsFile)
	if err != nil {
		fmt.Println(err)
	}
	settings := &settings{
		Navigation: "settings",
	}
	if err = json.Unmarshal(file, settings); err != nil {
		fmt.Println(err)
	}
	return settings
}

func contentType(imagePath string) string {
	imageType := path.Ext(imagePath)[1:]
	if imageType == "pdf" {
		return "application/pdf"
	}
	return fmt.Sprintf("image/%s", imageType)
}
