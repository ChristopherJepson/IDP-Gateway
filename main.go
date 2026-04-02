package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func enableCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. Tell the browser: "I explicitly allow requests from any origin"
		w.Header().Set("Access-Control-Allow-Origin", "*")

		// 2. Tell the browser: "I allow these specific types of interactions"
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		// 3. The Preflight Check: Browsers send an invisible "OPTIONS" request first
		// to test the waters. We just reply OK and stop here.
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// 4. If it passes the security check, move on to the actual function!
		next(w, r)
	}
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Method not allowed.", http.StatusMethodNotAllowed)
			return
		}

		// 1. Get the target format from the form (default to mp3 if missing)
		targetFormat := r.FormValue("targetFormat")
		if targetFormat == "" {
			targetFormat = "mp3"
		}

		r.ParseMultipartForm(25 << 20) // Increased to 25MB for larger WAV files

		file, header, err := r.FormFile("audioFile")
		if err != nil {
			http.Error(w, "Error retrieving file", http.StatusBadRequest)
			return
		}
		defer file.Close()

		inputPath := filepath.Join("temp", header.Filename)
		dst, _ := os.Create(inputPath)
		io.Copy(dst, file)
		dst.Close()

		// 2. Use the user's chosen target format for the output path
		fileNameWithoutExt := strings.TrimSuffix(header.Filename, filepath.Ext(header.Filename))
		outputFilename := fileNameWithoutExt + "." + targetFormat
		outputPath := filepath.Join("temp", outputFilename)

		fmt.Printf("Converting %s to %s...\n", header.Filename, targetFormat)

		// -y overwrites, -i is input. FFmpeg handles the codec based on extension.
		cmd := exec.Command("ffmpeg", "-y", "-i", inputPath, outputPath)

		err = cmd.Run()
		if err != nil {
			fmt.Printf("FFmpeg Error: %v\n", err)
			http.Error(w, "Conversion failed.", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"filename": "%s"}`, outputFilename)
	})(w, r)
}

func uploadVideoHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Method not allowed.", http.StatusMethodNotAllowed)
			return
		}

		// Set the limit to 100 MB
		const maxVideoSize = 100 << 20
		err := r.ParseMultipartForm(maxVideoSize)
		if err != nil {
			http.Error(w, "File too large. Max limit is 100MB.", http.StatusRequestEntityTooLarge)
			return
		}

		targetFormat := r.FormValue("targetFormat")
		file, header, err := r.FormFile("videoFile")
		if err != nil {
			http.Error(w, "Error retrieving file", http.StatusBadRequest)
			return
		}
		defer file.Close()

		inputPath := filepath.Join("temp", header.Filename)
		dst, _ := os.Create(inputPath)
		io.Copy(dst, file)
		dst.Close()

		fileNameWithoutExt := strings.TrimSuffix(header.Filename, filepath.Ext(header.Filename))
		outputFilename := fileNameWithoutExt + "." + targetFormat
		outputPath := filepath.Join("temp", outputFilename)

		fmt.Printf("Converting Video: %s to %s...\n", header.Filename, targetFormat)

		// FFmpeg Video Command
		// -preset ultrafast: Prioritizes speed over file size (important for responsiveness)
		cmd := exec.Command("ffmpeg", "-y", "-i", inputPath, "-preset", "ultrafast", outputPath)

		err = cmd.Run()
		if err != nil {
			fmt.Printf("Video Conversion Error: %v\n", err)
			http.Error(w, "Video conversion failed.", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"filename": "%s"}`, outputFilename)
	})(w, r)
}

func main() {
	os.MkdirAll("temp", os.ModePerm)

	// Wrap our existing endpoints in the new enableCORS middleware
	http.HandleFunc("/", enableCORS(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Your Go API Gateway is online!")
	}))

	http.HandleFunc("/upload", enableCORS(uploadHandler))

	// We handle the download folder slightly differently because it uses a built-in file server
	http.Handle("/download/", http.StripPrefix("/download/", http.HandlerFunc(enableCORS(func(w http.ResponseWriter, r *http.Request) {
		http.FileServer(http.Dir("temp")).ServeHTTP(w, r)
	}))))

	fmt.Println("Starting server on port 8080")
	http.ListenAndServe(":8080", nil)
}
