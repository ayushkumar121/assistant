//go:build darwin || linux

package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

func startAudioCapture() (string, error) {
	tmpFile := "/tmp/audio.wav"
	_ = os.Remove(tmpFile)

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("ffmpeg",
			"-f", "avfoundation", "-i", ":0",
			"-af", "silencedetect=noise=-30dB:d=2",
			"-ac", "1", "-ar", "16000", "-f", "wav", tmpFile)
	case "linux":
		cmd = exec.Command("ffmpeg",
			"-f", "alsa", "-i", "default",
			"-af", "silencedetect=noise=-30dB:d=2",
			"-ac", "1", "-ar", "16000", "-f", "wav", tmpFile)
	default:
		return "", fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	stdinPipe, _ := cmd.StdinPipe()
	stderrPipe, _ := cmd.StderrPipe()
	cmd.Stdout = io.Discard

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start ffmpeg: %v", err)
	}

	logger.Println("Recording started")

	done := make(chan struct{})
	go func() {
		scanner := bufio.NewScanner(stderrPipe)
		for scanner.Scan() {
			line := scanner.Text()
			debugLogger.Println(line)
			if strings.Contains(line, "silence_start") {
				logger.Println("Recording stopped")
				stdinPipe.Write([]byte("q\n")) // graceful stop
				break
			}
		}
		cmd.Wait()
		close(done)
	}()

	<-done

	// Ensure file is ready
	for range 20 {
		if fi, err := os.Stat(tmpFile); err == nil && fi.Size() > 1024 {
			return tmpFile, nil
		}
		time.Sleep(100 * time.Millisecond)
	}

	return "", fmt.Errorf("audio file was not created or too small")
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
