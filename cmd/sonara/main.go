/*
Setup:
  uv run scripts/fetch-headers.py
  uv run scripts/download-libs.py

Build:
  go build -o sonara ./cmd/sonara/

Download model:
  mkdir -p models
  wget -O models/ggml-tiny.bin https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-tiny.bin

Download sample:
  mkdir -p samples
  wget -O samples/jfk.wav https://github.com/ggml-org/whisper.cpp/raw/master/samples/jfk.wav

Run:
  ./sonara models/ggml-tiny.bin samples/jfk.wav
*/
package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"

	"github.com/thewh1teagle/sonara/internal/whisper"
)

var version = "dev"

func main() {
	if len(os.Args) == 2 && os.Args[1] == "--version" {
		fmt.Println(version)
		return
	}
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "usage: sonara <model.bin> <audio.wav>\n")
		os.Exit(1)
	}
	modelPath := os.Args[1]
	wavPath := os.Args[2]

	samples, err := readWAV(wavPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading wav: %v\n", err)
		os.Exit(1)
	}

	ctx, err := whisper.New(modelPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading model: %v\n", err)
		os.Exit(1)
	}
	defer ctx.Close()

	text, err := ctx.Transcribe(samples)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error transcribing: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(text)
}

// readWAV parses a 16-bit PCM WAV and returns float32 samples in [-1, 1].
// Assumes 16kHz mono — no resampling is done.
func readWAV(path string) ([]float32, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var header [12]byte
	if _, err := io.ReadFull(f, header[:]); err != nil {
		return nil, fmt.Errorf("failed to read WAV header: %w", err)
	}
	if string(header[0:4]) != "RIFF" || string(header[8:12]) != "WAVE" {
		return nil, fmt.Errorf("not a valid WAV file")
	}

	var audioFormat, channels, bitsPerSample uint16

	// Walk chunks to find fmt and data
	var dataSize uint32
	for {
		var chunkID [4]byte
		var chunkSize uint32
		if err := binary.Read(f, binary.LittleEndian, &chunkID); err != nil {
			return nil, fmt.Errorf("unexpected end of file: %w", err)
		}
		if err := binary.Read(f, binary.LittleEndian, &chunkSize); err != nil {
			return nil, fmt.Errorf("could not read chunk size: %w", err)
		}

		switch string(chunkID[:]) {
		case "fmt ":
			var fmtBuf [16]byte
			if _, err := io.ReadFull(f, fmtBuf[:]); err != nil {
				return nil, fmt.Errorf("failed to read fmt chunk: %w", err)
			}
			audioFormat = binary.LittleEndian.Uint16(fmtBuf[0:2])
			channels = binary.LittleEndian.Uint16(fmtBuf[2:4])
			bitsPerSample = binary.LittleEndian.Uint16(fmtBuf[14:16])
			// Skip any extra fmt bytes
			if chunkSize > 16 {
				if _, err := f.Seek(int64(chunkSize-16), io.SeekCurrent); err != nil {
					return nil, err
				}
			}
		case "data":
			dataSize = chunkSize
			goto readData
		default:
			if _, err := f.Seek(int64(chunkSize), io.SeekCurrent); err != nil {
				return nil, err
			}
		}
	}

readData:
	if audioFormat != 1 {
		return nil, fmt.Errorf("unsupported audio format %d (only PCM=1)", audioFormat)
	}
	if bitsPerSample != 16 {
		return nil, fmt.Errorf("unsupported bits per sample %d (only 16)", bitsPerSample)
	}

	nSamples := int(dataSize) / int(channels) / 2
	raw := make([]int16, int(dataSize)/2)
	if err := binary.Read(f, binary.LittleEndian, raw); err != nil {
		return nil, fmt.Errorf("failed to read PCM data: %w", err)
	}

	samples := make([]float32, nSamples)
	for i := 0; i < nSamples; i++ {
		var sum float64
		for ch := 0; ch < int(channels); ch++ {
			sum += float64(raw[i*int(channels)+ch])
		}
		samples[i] = float32(sum / float64(channels) / math.MaxInt16)
	}
	return samples, nil
}
