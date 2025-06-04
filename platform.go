package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
)

func recordAudio(filepath string, seconds int) error {
	fmt.Println("🎙️ Recording audio...")

	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("ffmpeg", "-f", "avfoundation", "-i", ":0", "-t", fmt.Sprintf("%d", seconds), "-ac", "1", "-ar", "16000", "-y", filepath)
	case "linux":
		cmd = exec.Command("ffmpeg", "-f", "alsa", "-i", "default", "-t", fmt.Sprintf("%d", seconds), "-ac", "1", "-ar", "16000", "-y", filepath)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stdin
	return cmd.Run()
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
