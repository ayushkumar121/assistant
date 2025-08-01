//go:build darwin || linux

package main

import (
	"bufio"
	"bytes"
	_ "embed"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
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
			"-af", "highpass=f=100,lowpass=f=3000",
			"-t", fmt.Sprint(duration),
			"-ac", "1", "-ar", "16000",
			"-c:a", "pcm_s16le", tmpFile)
	case "linux":
		cmd = exec.Command("ffmpeg",
			"-f", "alsa", "-i", "default",
			"-af", "highpass=f=100,lowpass=f=3000",
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
			"-af", "highpass=f=100,lowpass=f=3000,silencedetect=noise=-35dB:d=0.5",
			"-t", fmt.Sprint(maxAudioDuration),
			"-ac", "1", "-ar", "16000",
			"-c:a", "flac", tmpFile)
	case "linux":
		cmd = exec.Command("ffmpeg",
			"-f", "alsa", "-i", "default",
			"-af", "highpass=f=100,lowpass=f=3000,silencedetect=noise=-35dB:d=0.5",
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
		var timer *time.Timer
		scanner := bufio.NewScanner(stderrPipe)
		for scanner.Scan() {
			line := scanner.Text()
			debugLogger.Println(line)

			if strings.Contains(line, "silence_start") {
				if timer != nil {
					timer.Stop()
				}
				timer = time.AfterFunc(maxSilenceDuration*time.Second, func() {
					stdinPipe.Write([]byte("q\n"))
					logger.Println("Recording stopped")
				})
			} else if strings.Contains(line, "silence_end") {
				if timer != nil {
					timer.Stop()
					timer = nil
				}
			}
		}
	}()

	// Wait until the command exits
	if err := cmd.Wait(); err != nil {
		return "", fmt.Errorf("ffmpeg exited with error: %v", err)
	}

	return tmpFile, nil
}

func speakFromReader(r io.Reader) error {
	ffplayFlags := []string{
		"-autoexit",
		"-nodisp",
		"-af", "adelay=500|500:all=1",
		"-i", "pipe:0",
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command(resolveExecutablePath("ffplay"), ffplayFlags...)
	case "linux":
		cmd = exec.Command("ffplay", ffplayFlags...)
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

func playAudio(data []byte) error {
	r := bytes.NewReader(data)

	ffplayFlags := []string{
		"-autoexit",
		"-nodisp",
		"-i", "pipe:0",
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command(resolveExecutablePath("ffplay"), ffplayFlags...)
	case "linux":
		cmd = exec.Command("ffplay", ffplayFlags...)
	}

	cmd.Stdin = r
	if DebugEnabled() {
		cmd.Stderr = os.Stderr
	} else {
		cmd.Stderr = io.Discard
	}

	return cmd.Run()
}
