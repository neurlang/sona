package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gen2brain/malgo"
	"github.com/go-audio/audio"
	"github.com/go-audio/wav"
)


// recordFromMicrophone captures audio from the default microphone and saves it as a WAV file
func recordFromMicrophone() (string, error) {
	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "recording-*.wav")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %v", err)
	}
	tmpFile.Close() // We'll reopen it later for writing

	// Initialize malgo
	ctx, err := malgo.InitContext(nil, malgo.ContextConfig{}, func(message string) {
		// Optional logging callback
	})
	if err != nil {
		return "", fmt.Errorf("failed to initialize context: %v", err)
	}
	defer ctx.Uninit()

	// Configure capture device
	deviceConfig := malgo.DefaultDeviceConfig(malgo.Capture)
	deviceConfig.Capture.Format = malgo.FormatS16
	deviceConfig.Capture.Channels = 1
	deviceConfig.SampleRate = 16000
	deviceConfig.Alsa.NoMMap = 1 // Fix for some Linux systems

	// Prepare for recording
	var capturedSamples []byte
	stopRecording := make(chan struct{})
	recordingDone := make(chan struct{})

	// Callback for captured frames
	onRecvFrames := func(pSample2, pSample []byte, framecount uint32) {
		select {
		case <-stopRecording:
			// Don't append more samples if we're stopping
			return
		default:
			capturedSamples = append(capturedSamples, pSample...)
		}
	}

	// Create and start capture device
	captureCallbacks := malgo.DeviceCallbacks{
		Data: onRecvFrames,
	}
	
	device, err := malgo.InitDevice(ctx.Context, deviceConfig, captureCallbacks)
	if err != nil {
		return "", fmt.Errorf("failed to initialize device: %v", err)
	}
	defer device.Uninit()

	err = device.Start()
	if err != nil {
		return "", fmt.Errorf("failed to start device: %v", err)
	}

	// Recording has started
	fmt.Println("[Press Enter to stop]")

	// Wait for Enter in a goroutine
	go func() {
		fmt.Scanln()
		close(stopRecording)
		
		// Give a moment for the last samples to be processed
		time.Sleep(100 * time.Millisecond)
		device.Stop()
		close(recordingDone)
	}()

	// Wait for recording to finish
	<-recordingDone

	// Convert captured samples to WAV
	err = saveAsWAV(tmpFile.Name(), capturedSamples, 16000)
	if err != nil {
		return "", fmt.Errorf("failed to save WAV file: %v", err)
	}

	return tmpFile.Name(), nil
}

// saveAsWAV converts raw PCM samples to a proper WAV file
func saveAsWAV(filename string, samples []byte, sampleRate int) error {
	// Convert bytes to int slice for go-audio
	intData := make([]int, len(samples)/2)
	for i := 0; i < len(samples); i += 2 {
		if i+1 < len(samples) {
			// Little-endian int16 to int
			value := int(int16(samples[i]) | int16(samples[i+1])<<8)
			intData[i/2] = value
		}
	}

	// Create audio buffer
	audioBuf := &audio.IntBuffer{
		Data: intData,
		Format: &audio.Format{
			SampleRate:  sampleRate,
			NumChannels: 1,
		},
	}

	// Create and write WAV file
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := wav.NewEncoder(file, sampleRate, 16, 1, 1)
	defer encoder.Close()

	return encoder.Write(audioBuf)
}

func main() {
	// Define command line flags
	port := flag.String("port", "36055", "Port number for the API server")
	filePath := flag.String("file", "", "Path to the WAV file")
	flag.Parse()

	// If no file specified, record from microphone
	if *filePath == "" {
		// Wait for Enter to start
		fmt.Print("[Press Enter to start recording]")
		fmt.Scanln()
		
		// Record from microphone
		recordedFile, err := recordFromMicrophone()
		if err != nil {
			return
		}
		defer os.Remove(recordedFile) // Clean up temp file
		
		*filePath = recordedFile
		//fmt.Printf("Recording saved to temp file: %s\n", recordedFile)
	}

	// Construct the base URL with configurable port
	baseURL := fmt.Sprintf("http://127.0.0.1:%s", *port)
	apiEndpoint := "/v1/audio/transcriptions"
	fullURL := baseURL + apiEndpoint

	// Open the file
	file, err := os.Open(*filePath)
	if err != nil {
		fmt.Printf("Error opening file: %v\n", err)
		return
	}
	defer file.Close()

	// Create a buffer to write our multipart form
	var requestBody bytes.Buffer
	multipartWriter := multipart.NewWriter(&requestBody)

	// Add the file to the multipart form
	fileWriter, err := multipartWriter.CreateFormFile("file", filepath.Base(*filePath))
	if err != nil {
		fmt.Printf("Error creating form file: %v\n", err)
		return
	}
	
	// Copy the file content to the multipart section
	_, err = io.Copy(fileWriter, file)
	if err != nil {
		fmt.Printf("Error copying file content: %v\n", err)
		return
	}

	// Close the multipart writer to set the terminating boundary
	multipartWriter.Close()

	// Create the HTTP request
	req, err := http.NewRequest("POST", fullURL, &requestBody)
	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		return
	}

	// Set the content type header to multipart form data with boundary
	req.Header.Set("Content-Type", multipartWriter.FormDataContentType())

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error sending request: %v\n", err)
		return
	}
	defer resp.Body.Close()

	// Read the response
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response: %v\n", err)
		return
	}

	// Check if the request was successful
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Error: API returned status %s\n", resp.Status)
		fmt.Printf("Response: %s\n", string(responseBody))
		return
	}

	// Parse the JSON response to extract the text field
	var result map[string]interface{}
	err = json.Unmarshal(responseBody, &result)
	if err != nil {
		fmt.Printf("Error parsing JSON response: %v\n", err)
		fmt.Printf("Raw response: %s\n", string(responseBody))
		return
	}

	// Extract and print only the text field
	if text, ok := result["text"]; ok {
		fmt.Printf("%v\n", text)
	} else {
		fmt.Println("Error: 'text' field not found in response")
		fmt.Printf("Full response: %s\n", string(responseBody))
	}
}
