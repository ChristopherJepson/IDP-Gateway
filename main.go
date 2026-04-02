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
	cmd := exec.Command("ffmpeg", "-y", "-i", inputPath, outputPath) // Added -y to automatically overwrite if file exists

	err = cmd.Run()
	if err != nil {
		fmt.Printf("FFmpeg Error: %v\n", err)
		http.Error(w, "Failed to convert file.", http.StatusInternalServerError)
		return
	}

	fmt.Printf("Success! Converted file saved to: %s\n", outputPath)

	// NEW LOGIC: Send back a clickable HTML link instead of plain text
	w.Header().Set("Content-Type", "text/html")
	downloadURL := fmt.Sprintf("http://192.168.0.15:8080/download/%s.mp3", fileNameWithoutExt)

	htmlResponse := fmt.Sprintf(`
		<h2>Conversion Successful!</h2>
		<p>Your file has been converted to MP3.</p>
		<a href="%s" style="padding: 10px 20px; background-color: #4CAF50; color: white; text-decoration: none; border-radius: 5px;">Download %s.mp3</a>
		<br><br>
		<a href="javascript:history.back()">Convert another file</a>
	`, downloadURL, fileNameWithoutExt)

	fmt.Fprint(w, htmlResponse)
}

func main() {
	os.MkdirAll("temp", os.ModePerm)

	// NEW LOGIC: Create a file server that securely serves files from the "temp" directory
	http.Handle("/download/", http.StripPrefix("/download/", http.FileServer(http.Dir("temp"))))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Your Go API Gateway is online!")
	})
	http.HandleFunc("/upload", uploadHandler)

	fmt.Println("Starting server on port 8080")
	http.ListenAndServe(":8080", nil)
}
