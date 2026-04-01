package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec" // NEW: Allows Go to run terminal commands
	"path/filepath"
	"strings" // NEW: Allows us to manipulate text strings
)

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	// 1. Security & Size Checks
	if r.Method != "POST" {
		http.Error(w, "Method not allowed.", http.StatusMethodNotAllowed)
		return
	}
	r.ParseMultipartForm(10 << 20)

	// 2. Retrieve the file
	file, header, err := r.FormFile("audioFile")
	if err != nil {
		http.Error(w, "Error retrieving file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// 3. Save the original file to disk
	inputPath := filepath.Join("temp", header.Filename)
	dst, _ := os.Create(inputPath)
	io.Copy(dst, file)
	dst.Close() // Close it immediately so FFmpeg can access it

	// --------------------------------------------------------
	// NEW LOGIC: The FFmpeg Orchestrator
	// --------------------------------------------------------

	// 4. Figure out what the new file should be called (change extension to .mp3)
	// Example: "song.wav" becomes "song" + ".mp3"
	fileNameWithoutExt := strings.TrimSuffix(header.Filename, filepath.Ext(header.Filename))
	outputPath := filepath.Join("temp", fileNameWithoutExt+".mp3")

	// 5. Tell Go to run the FFmpeg command
	fmt.Printf("Starting conversion for: %s...\n", header.Filename)
	cmd := exec.Command("ffmpeg", "-i", inputPath, outputPath)

	// 6. Execute the command and capture any errors
	err = cmd.Run()
	if err != nil {
		fmt.Printf("FFmpeg Error: %v\n", err)
		http.Error(w, "Failed to convert file. Is FFmpeg installed?", http.StatusInternalServerError)
		return
	}

	// 7. Success!
	fmt.Printf("Success! Converted file saved to: %s\n", outputPath)
	fmt.Fprintf(w, "API Gateway successfully converted your file to: %s.mp3", fileNameWithoutExt)
}

func main() {
	// Ensure the temp directory exists before the server starts
	os.MkdirAll("temp", os.ModePerm)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Your Go API Gateway is online!")
	})
	http.HandleFunc("/upload", uploadHandler)

	fmt.Println("Starting server on http://localhost:8080")
	http.ListenAndServe(":8080", nil)
}
