//go:build darwin || linux

package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
)

func startAudioCapture() (string, error) {
	tmpFile := "/tmp/audio.flac"
	_ = os.Remove(tmpFile)

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command(resolveExecutablePath("ffmpeg"),
			"-f", "avfoundation", "-i", ":0",
			"-af", "silenceremove=stop_periods=-1:stop_duration=1:stop_threshold=-40dB",
			"-t", fmt.Sprint(maxAudioDuration),
			"-ac", "1", "-ar", "16000",
			"-c:a", "flac", tmpFile)
	case "linux":
		cmd = exec.Command("ffmpeg",
			"-f", "alsa", "-i", "default",
			"-af", "silenceremove=stop_periods=-1:stop_duration=1:stop_threshold=-40dB",
			"-t", fmt.Sprint(maxAudioDuration),
			"-ac", "1", "-ar", "16000",
			"-c:a", "flac", tmpFile)
	default:
		return "", fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	stdinPipe, _ := cmd.StdinPipe()
	if DebugEnabled() {
		cmd.Stderr = os.Stderr
	} else {
		cmd.Stderr = io.Discard
	}
	cmd.Stdout = io.Discard

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start ffmpeg: %v", err)
	}

	logger.Println("Recording started. Press any Enter to stop...")

	go func() {
		bufio.NewReader(os.Stdin).ReadByte()
		stdinPipe.Write([]byte("q\n"))
		logger.Println("Recording stopped")
	}()

	// Wait until the command exits
	cmd.Wait()

	return tmpFile, nil
}

// speakFromReader runs the platform-specific audio player and streams from r
func speakFromReader(r io.Reader) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command(resolveExecutablePath("ffplay"), "-autoexit", "-")
	case "linux":
		cmd = exec.Command("ffplay", "-autoexit", "-")
	}
	cmd.Stdin = r
	if DebugEnabled() {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	} else {
		cmd.Stdout = io.Discard
		cmd.Stderr = io.Discard
	}
	logger.Println("Speaking...")
	return cmd.Run()
}
