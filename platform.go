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

func recordAudio(duration int) (string, error) {
	logger.Println("Listening...")
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

	err := cmd.Run()
	if err != nil {
		return "", err
	}
	return tmpFile, nil
}

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
	logger.Println("Speaking...")
	return cmd.Run()
}
