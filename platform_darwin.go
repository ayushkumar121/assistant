//go:build darwin

package main

import (
	"fmt"
	"os/exec"
)

func recordAudio(filepath string, seconds int) error {
	fmt.Println("🎙️ Recording audio (macOS)...")
	cmd := exec.Command("ffmpeg", "-f", "avfoundation", "-i", ":0", "-t", fmt.Sprintf("%d", seconds), "-ac", "1", "-ar", "16000", "-y", filepath)
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run()
}

func playAudio(path string) error {
	fmt.Println("🔊 Playing audio (macOS)...")
	cmd := exec.Command("afplay", path)
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run()
}
