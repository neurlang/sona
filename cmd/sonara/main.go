package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/thewh1teagle/sonara/internal/audio"
	"github.com/thewh1teagle/sonara/internal/server"
	"github.com/thewh1teagle/sonara/internal/whisper"
)

var version = "dev"

func main() {
	rootCmd := &cobra.Command{
		Use:     "sonara",
		Short:   "Speech-to-text powered by whisper.cpp",
		Version: version,
	}

	transcribeCmd := &cobra.Command{
		Use:   "transcribe <model.bin> <audio.wav>",
		Short: "Transcribe an audio file",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			modelPath := args[0]
			wavPath := args[1]

			samples, err := audio.ReadFile(wavPath)
			if err != nil {
				return fmt.Errorf("error reading audio: %w", err)
			}

			ctx, err := whisper.New(modelPath)
			if err != nil {
				return fmt.Errorf("error loading model: %w", err)
			}
			defer ctx.Close()

			text, err := ctx.Transcribe(samples)
			if err != nil {
				return fmt.Errorf("error transcribing: %w", err)
			}
			fmt.Println(text)
			return nil
		},
	}

	var port int
	serveCmd := &cobra.Command{
		Use:   "serve <model.bin>",
		Short: "Start an OpenAI-compatible transcription server",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			modelPath := args[0]

			ctx, err := whisper.New(modelPath)
			if err != nil {
				return fmt.Errorf("error loading model: %w", err)
			}
			defer ctx.Close()

			srv := server.New(ctx, modelPath)
			addr := fmt.Sprintf(":%d", port)
			return server.ListenAndServe(addr, srv)
		},
	}
	serveCmd.Flags().IntVarP(&port, "port", "p", 11531, "port to listen on")

	rootCmd.AddCommand(transcribeCmd, serveCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
