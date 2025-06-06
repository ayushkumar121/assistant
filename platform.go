//go:build darwin || linux

package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
)

// startAudioCapture starts ffmpeg and returns a pipe of raw WAV audio
func startAudioCapture(duration int) (io.ReadCloser, error) {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("ffmpeg",
			"-f", "avfoundation", "-i", ":0",
			"-t", fmt.Sprintf("%d", duration),
			"-ac", "1", "-ar", "16000", "-f", "wav", "-")

	case "linux":
		cmd = exec.Command("ffmpeg",
			"-f", "alsa", "-i", "default",
			"-t", fmt.Sprintf("%d", duration),
			"-ac", "1", "-ar", "16000", "-f", "wav", "-")

	default:
		return nil, fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("ffmpeg stdout error: %v", err)
	}
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start ffmpeg: %v", err)
	}

	return stdout, nil
}

// speakFromReader runs the platform-specific audio player and streams from r
func speakFromReader(r io.Reader) error {
	cmd := exec.Command("mpg123", "-")
	cmd.Stdin = r
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Println("🔊 Speaking...")
	return cmd.Run()
}
