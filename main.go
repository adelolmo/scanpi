package main

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/adelolmo/scanpi/debug"
	"github.com/adelolmo/scanpi/fsutils"
	"github.com/adelolmo/scanpi/graphic"
	"github.com/adelolmo/scanpi/pdf"
	"github.com/adelolmo/scanpi/zipper"
	"github.com/gorilla/mux"
	"html/template"
	"io/fs"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
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
	Scans        []image
	JobStarted   bool
}

type image struct {
	Name     string
	LinkName string
}

type pageScanner struct {
	Navigation string
	JobName    string
	Scans      []image
	JobStarted bool
}

type configuration struct {
	OutputDirectory string
	WorkDirectory   string
	ThumbnailFilter string
}

//go:embed assets templates/*
var content embed.FS

var indexTemplate *template.Template
var jobTemplate *template.Template
var jobsTemplate *template.Template
var settingsTemplate *template.Template

var appConfiguration configuration
var thumb *graphic.Thumbnail

func main() {
	indexTemplate = template.Must(template.ParseFS(content, "templates/index.html", "templates/header.html"))
	jobTemplate = template.Must(template.ParseFS(content, "templates/job.html", "templates/header.html"))
	jobsTemplate = template.Must(template.ParseFS(content, "templates/jobs.html", "templates/header.html"))
	settingsTemplate = template.Must(template.ParseFS(content, "templates/settings.html", "templates/header.html"))

	port := os.Getenv("port")
	if port == "" {
		port = "8000"
	}
	outputDirectory := os.Getenv("output_dir")
	migrate(outputDirectory)
	workDirectory := os.Getenv("work_dir")
	thumbnailFilter := os.Getenv("thumbnail_filter")
	appConfiguration = configuration{
		OutputDirectory: outputDirectory,
		WorkDirectory:   workDirectory,
		ThumbnailFilter: thumbnailFilter,
	}
	fmt.Println(fmt.Sprintf("port: %s, output_dir: %s, work_dir: %s, thumbnail_filter: %s debug: %v",
		port, outputDirectory, workDirectory, thumbnailFilter, debug.Enabled()))

	settingsFile := path.Join(appConfiguration.WorkDirectory, "settings.json")
	if _, err := os.Stat(settingsFile); os.IsNotExist(err) {
		settings := &settings{
			Mode:       graphic.Color.String(),
			Format:     graphic.Jpeg.String(),
			Resolution: "200",
		}
		settingsJson, _ := json.Marshal(settings)
		if err := ioutil.WriteFile(path.Join(appConfiguration.WorkDirectory, "settings.json"), settingsJson, 0644); err != nil {
			log.Fatalln(err)
		}
	}

	thumb = graphic.NewThumbnail(appConfiguration.ThumbnailFilter,
		appConfiguration.OutputDirectory)

	router := mux.NewRouter()
	fsys, err := fs.Sub(content, "assets")
	if err != nil {
		log.Fatal(err)
	}
	router.PathPrefix("/assets/").Handler(http.StripPrefix("/assets/", http.FileServer(http.FS(fsys))))
	router.HandleFunc("/", homePage).Methods("GET")
	router.HandleFunc("/settings", showSettingsPage).Methods("GET")
	router.HandleFunc("/settings", updateSettingsPage).Methods("POST")
	router.HandleFunc("/jobs", showJobsPage).Methods("GET")
	router.HandleFunc("/job", resumeJobPage).Methods("GET")
	router.HandleFunc("/job", createJobHandler).Methods("POST")
	router.HandleFunc("/deleteJob", deleteJobHandler).Methods("POST")
	router.HandleFunc("/renameJob", renameJobHandler).Methods("POST")
	router.HandleFunc("/scan", scanHandler).Methods("POST")
	router.HandleFunc("/deleteScan", deleteScanHandler).Methods("POST")
	router.HandleFunc("/download", downloadFileHandler).Methods("GET")
	router.HandleFunc("/image", imageHandler).Methods("GET")
	router.HandleFunc("/downloadall", downloadAllHandler).Methods("GET")
	router.HandleFunc("/preview", previewHandler).Methods("GET")

	router.HandleFunc("/scanner", scannerHandler).Methods("GET")

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), router))
}

func homePage(w http.ResponseWriter, r *http.Request) {
	var previousJobs []string
	for _, dir := range fsutils.JobDirectories(appConfiguration.OutputDirectory) {
		previousJobs = append(previousJobs, dir.Name())
	}

	type index struct {
		Navigation string
	}

	w.Header().Add("Content-Type", "text/html")
	if err := indexTemplate.Execute(w, &index{Navigation: "home"}); err != nil {
		fmt.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func showSettingsPage(w http.ResponseWriter, r *http.Request) {
	settings := readSettings()
	w.Header().Add("Content-Type", "text/html")
	if err := settingsTemplate.Execute(w, settings); err != nil {
		fmt.Println(err)
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

	w.Header().Add("Content-Type", "text/html")
	if err := settingsTemplate.Execute(w, settings); err != nil {
		fmt.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func showJobsPage(w http.ResponseWriter, r *http.Request) {
	var previousJobs []string
	for _, dir := range fsutils.JobDirectories(appConfiguration.OutputDirectory) {
		previousJobs = append(previousJobs, dir.Name())
	}

	index := &pageJobs{
		Navigation:   "jobs",
		PreviousJobs: previousJobs,
	}

	w.Header().Add("Content-Type", "text/html")
	if err := jobsTemplate.Execute(w, index); err != nil {
		fmt.Println(err)
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

	scans, err := listJobImages(jobName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	scanner := &pageJobs{
		Navigation: "jobs",
		JobName:    jobName,
		Scans:      scans,
	}

	w.Header().Add("Content-Type", "text/html")
	if err := jobTemplate.Execute(w, scanner); err != nil {
		fmt.Println(err)
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

	scans, err := listJobImages(jobName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	scanner := &pageScanner{
		Navigation: "jobs",
		Scans:      scans,
		JobName:    jobName,
	}

	w.Header().Add("Content-Type", "text/html")
	if err := jobTemplate.Execute(w, scanner); err != nil {
		fmt.Println(err)
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
	for _, dir := range fsutils.JobDirectories(appConfiguration.OutputDirectory) {
		previousJobs = append(previousJobs, dir.Name())
	}

	index := &pageJobs{
		Navigation:   "jobs",
		PreviousJobs: previousJobs,
	}

	w.Header().Add("Content-Type", "text/html")
	if err := jobsTemplate.Execute(w, index); err != nil {
		fmt.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func renameJobHandler(w http.ResponseWriter, r *http.Request) {
	currentJobName := r.FormValue("currentJobName")
	newJobName := r.FormValue("newJobName")

	currentJobPath := path.Join(appConfiguration.OutputDirectory, currentJobName)
	newJobPath := path.Join(appConfiguration.OutputDirectory, newJobName)

	w.Header().Set("Location", "/job?jobName="+newJobName)

	err := os.Rename(currentJobPath, newJobPath)
	if err != nil {
		if errors.Unwrap(err) == syscall.EEXIST {
			w.WriteHeader(303)
			return
		}
		fmt.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(303)
}

func scanHandler(w http.ResponseWriter, r *http.Request) {
	jobName := r.FormValue("jobName")
	previousScans, err := listJobImages(jobName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	linkName := "1"
	if len(previousScans) > 0 {
		lastScanName := previousScans[len(previousScans)-1].LinkName
		lastScanNumber, err := strconv.Atoi(strings.Split(lastScanName, ".")[0])
		if err != nil {
			fmt.Println(err)
		}
		linkName = strconv.Itoa(lastScanNumber + 1)
	}

	settings := readSettings()

	resolution, _ := strconv.Atoi(settings.Resolution)
	scanJob := graphic.NewScanJob(
		graphic.ToMode(settings.Mode),
		graphic.ToFormat(settings.Format),
		resolution,
		thumb,
	)
	imageDetails := graphic.ImageDetails{
		Name:          fsutils.GenerateDateFilename(),
		LinkName:      linkName,
		Format:        graphic.ToFormat(readSettings().Format),
		Directory:     jobName,
		BaseDirectory: appConfiguration.OutputDirectory,
	}
	scanJob.StartScanning(imageDetails)

	var scans []image
	for _, file := range previousScans {
		scans = append(scans, file)
	}

	scanner := &pageJobs{
		Navigation: "jobs",
		JobName:    jobName,
		Scans:      scans,
		JobStarted: true,
	}

	w.Header().Add("Content-Type", "text/html")
	if err := jobTemplate.Execute(w, scanner); err != nil {
		fmt.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func deleteScanHandler(w http.ResponseWriter, r *http.Request) {
	jobName := r.FormValue("jobName")
	scan := r.FormValue("scan")
	imagePath := path.Join(appConfiguration.OutputDirectory, jobName, scan)

	debug.Info(fmt.Sprintf("delete image %s\n", imagePath))

	readlink, err := os.Readlink(imagePath)
	if err != nil {
		fmt.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// delete image
	if err := os.Remove(path.Join(appConfiguration.OutputDirectory, jobName, readlink)); err != nil {
		fmt.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// delete symlink
	if err := os.Remove(imagePath); err != nil {
		fmt.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := thumb.DeletePreview(imagePath); err != nil {
		fmt.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	scans, err := listJobImages(jobName)
	if err != nil {
		fmt.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	scanner := &pageJobs{
		Navigation: "jobs",
		JobName:    jobName,
		Scans:      scans,
	}

	w.Header().Add("Content-Type", "text/html")
	if err := jobTemplate.Execute(w, scanner); err != nil {
		fmt.Println(err)
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

	image, contentType, err := readImage(jobName, scan)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", strconv.Itoa(len(image)))
	w.Header().Set("content-disposition",
		fmt.Sprintf("attachment; filename=\"%s-%s\"", jobName, scan))
	if _, err := w.Write(image); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func imageHandler(w http.ResponseWriter, r *http.Request) {
	debug.Info("imageHandler")
	encodedJobName := r.FormValue("jobName")
	jobName, err := url.QueryUnescape(encodedJobName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	scan := r.FormValue("scan")

	image, contentType, err := readImage(jobName, scan)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", strconv.Itoa(len(image)))
	if _, err := w.Write(image); err != nil {
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

	scans, err := listJobImages(jobName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	switch envelope {
	case "zip":
		w.Header().Set("content-disposition", fmt.Sprintf("attachment; filename=\"%s.zip\"", jobName))
		zip := zipper.NewZipper(w)
		for _, scanImage := range scans {
			if err := zip.AddFile(path.Join(appConfiguration.OutputDirectory, jobName, scanImage.Name), scanImage.Name); err != nil {
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
		for _, scanImage := range scans {
			if err := pdfFile.AddImage(path.Join(appConfiguration.OutputDirectory, jobName, scanImage.Name)); err != nil {
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
	debug.Info("previewHandler")
	encodedJobName := r.FormValue("jobName")
	jobName, err := url.QueryUnescape(encodedJobName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	scan := r.FormValue("scan")

	imagePath := path.Join(appConfiguration.OutputDirectory, jobName, scan)
	buffer, err := thumb.Preview(imagePath)
	if err != nil {
		fsys, err := fs.Sub(content, "assets")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		file, err := fs.ReadFile(fsys, "not_available.jpeg")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if _, err := w.Write(file); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "image/jpeg")
		w.Header().Set("Content-Length", strconv.Itoa(len(file)))
		return
	}

	w.Header().Set("Content-Type", "image/jpeg")
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
	deviceName, err := graphic.ScannerDevice()
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

func readImage(jobName string, scan string) ([]byte, string, error) {
	imagePath := path.Join(appConfiguration.OutputDirectory, jobName, scan)
	debug.Info(fmt.Sprintf("imagePath: %s", imagePath))
	file, err := ioutil.ReadFile(imagePath)
	if err != nil {
		return nil, "", err
	}
	return file, contentType(imagePath), nil
}

func contentType(imagePath string) string {
	imageType := path.Ext(imagePath)[1:]
	if imageType == "pdf" {
		return "application/pdf"
	}
	return fmt.Sprintf("image/%s", imageType)
}

func listJobImages(jobName string) ([]image, error) {
	var scans []image
	fileMetaDataSlice, err := fsutils.ImageFilesOnDirectory(path.Join(appConfiguration.OutputDirectory, jobName))
	if err != nil {
		return []image{}, err
	}
	for _, file := range fileMetaDataSlice {
		scans = append(scans, image{
			Name:     file.Filename,
			LinkName: file.LinkName,
		})
	}
	return scans, nil
}

// TODO delete
func migrate(baseDir string) {
	files, err := ioutil.ReadDir(baseDir)
	if err != nil {
		debug.Error(fmt.Sprintf("unable to get directories from '%s'", baseDir))
	}
	sort.Slice(files, func(i, j int) bool {
		return files[j].ModTime().After(files[i].ModTime())
	})

	for _, file := range files {
		if !file.IsDir() {
			continue
		}
		migrateJob(path.Join(baseDir, file.Name()))
	}
	debug.Info("Migration finished")
}

func migrateJob(dir string) {
	debug.Info(fmt.Sprintf("Migrating directory job: %s", dir))
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		debug.Error(fmt.Sprintf("unable to get images from directory '%s'", dir))
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		if len(file.Name()) < 16 && file.Mode()&os.ModeSymlink != 0 {
			continue
		}
		ext := path.Ext(file.Name())
		if ext != ".tiff" && ext != ".png" && ext != ".jpeg" && ext != ".pnm" && ext != ".pdf" {
			continue
		}
		if len(file.Name()) == 19 {
			continue
		}

		filename := fsutils.GenerateDateFilename() + ext
		thumbnailFilename := filename + ".thumbnail"
		linkFilename := file.Name()
		thumbnailLinkName := file.Name() + ".thumbnail"

		// images
		err := os.Rename(path.Join(dir, linkFilename), path.Join(dir, filename))
		if err != nil {
			debug.Error(fmt.Sprintf("unable to rename file '%s' to '%s'", linkFilename, filename))
		}

		err = os.Symlink(filename, path.Join(dir, linkFilename))
		if err != nil {
			debug.Error(fmt.Sprintf("Cannot create symlink to image file on '%s'. Error: %s", filename, err))
		}

		// thumbnails
		err = os.Rename(path.Join(dir, thumbnailLinkName), path.Join(dir, thumbnailFilename))
		if err != nil {
			debug.Error(fmt.Sprintf("unable to rename file '%s' to '%s'", thumbnailLinkName, thumbnailFilename))
		}

		err = os.Symlink(thumbnailFilename, path.Join(dir, thumbnailLinkName))
		if err != nil {
			debug.Error(fmt.Sprintf("Cannot create symlink to image file on '%s'. Error: %s", filename, err))
		}

		time.Sleep(1100 * time.Millisecond)
	}
}
