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

// NEW LOGIC: The Security Middleware
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
	if r.Method != "POST" {
		http.Error(w, "Method not allowed.", http.StatusMethodNotAllowed)
		return
	}
	r.ParseMultipartForm(10 << 20)

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

	fileNameWithoutExt := strings.TrimSuffix(header.Filename, filepath.Ext(header.Filename))
	outputPath := filepath.Join("temp", fileNameWithoutExt+".mp3")

	fmt.Printf("Starting conversion for: %s...\n", header.Filename)
	cmd := exec.Command("ffmpeg", "-y", "-i", inputPath, outputPath)

	err = cmd.Run()
	if err != nil {
		fmt.Printf("FFmpeg Error: %v\n", err)
		http.Error(w, "Failed to convert file.", http.StatusInternalServerError)
		return
	}

	fmt.Printf("Success! Converted file saved to: %s\n", outputPath)

	// Since we are talking to a React API now instead of a raw HTML page,
	// we just need to send a simple success code back.
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Converted successfully")
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
