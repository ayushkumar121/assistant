//go:build linux

package main

import (
	"fmt"
	"os/exec"
)

func recordAudio(filepath string, seconds int) error {
	fmt.Println("🎙️ Recording audio (Linux)...")
	cmd := exec.Command("ffmpeg", "-f", "alsa", "-i", "default", "-t", fmt.Sprintf("%d", seconds), "-ac", "1", "-ar", "16000", "-y", filepath)
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run()
}

func playAudio(path string) error {
	fmt.Println("🔊 Playing audio (Linux)...")
	cmd := exec.Command("mpg123", path)
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run()
}
