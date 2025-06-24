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
)

func recordAudio(duration int) (string, error) {
	logger.Println("Recording audio")
	tmpFile := "/tmp/recording.wav"
	_ = os.Remove(tmpFile)

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command(resolveExecutablePath("ffmpeg"),
			"-f", "avfoundation", "-i", ":0",
			"-t", fmt.Sprint(duration),
			"-ac", "1", "-ar", "16000",
			"-c:a", "pcm_s16le", tmpFile)
	case "linux":
		cmd = exec.Command("ffmpeg",
			"-f", "alsa", "-i", "default",
			"-t", fmt.Sprint(duration),
			"-ac", "1", "-ar", "16000",
			"-c:a", "pcm_s16le", tmpFile)
	default:
		return "", fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	if DebugEnabled() {
		cmd.Stderr = os.Stderr
	} else {
		cmd.Stderr = io.Discard
	}

	err := cmd.Run()
	if err != nil {
		return "", err
	}
	return tmpFile, nil
}

func startAudioCapture() (string, error) {
	logger.Println("Starting audio capture")
	tmpFile := "/tmp/audio.flac"
	_ = os.Remove(tmpFile)

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command(resolveExecutablePath("ffmpeg"),
			"-f", "avfoundation", "-i", ":0",
			"-af", "silencedetect=noise=-50dB:d=2",
			"-t", fmt.Sprint(maxAudioDuration),
			"-ac", "1", "-ar", "16000",
			"-c:a", "flac", tmpFile)
	case "linux":
		cmd = exec.Command("ffmpeg",
			"-f", "alsa", "-i", "default",
			"-af", "silencedetect=noise=-50dB:d=2",
			"-t", fmt.Sprint(maxAudioDuration),
			"-ac", "1", "-ar", "16000",
			"-c:a", "flac", tmpFile)
	default:
		return "", fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
	stdinPipe, _ := cmd.StdinPipe()
	stderrPipe, _ := cmd.StderrPipe()

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start ffmpeg: %v", err)
	}

	go func() {
		scanner := bufio.NewScanner(stderrPipe)
		for scanner.Scan() {
			line := scanner.Text()
			debugLogger.Println(line)
			if strings.Contains(line, "silence_end") {
				stdinPipe.Write([]byte("q\n"))
				logger.Println("Recording stopped")
				break
			}
		}
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
		cmd.Stderr = os.Stderr
	} else {
		cmd.Stderr = io.Discard
	}
	logger.Println("Speaking...")
	return cmd.Run()
}
