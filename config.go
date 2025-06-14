package main

import (
	"io"
	"log"
	"os"
	"path/filepath"
)

var (
	OpenAIAPIKey string
	DebugMode    string
)

const (
	memoryFile       = "memory.txt"
	memoryLimit      = 50
	chatHistoryLimit = 20
	maxAudioDuration = 12
	whisperURL       = "https://api.openai.com/v1/audio/transcriptions"
	chatURL          = "https://api.openai.com/v1/chat/completions"
	ttsURL           = "https://api.openai.com/v1/audio/speech"

	whisperModel = "whisper-1"
	chatModel    = "gpt-4.1"
	ttsModel     = "tts-1"
	ttsVoice     = "alloy"
)

var debugLogger = log.New(io.Discard, "DEBUG ", log.LstdFlags)
var logger = log.New(os.Stderr, "INFO ", log.LstdFlags)

func DebugEnabled() bool {
	return DebugMode != ""
}

func init() {
	if DebugEnabled() {
		debugLogger.SetOutput(os.Stderr)
	}
}

func resolveExecutablePath(name string) string {
	programPath, err := os.Executable()
	if err != nil {
		panic(err)
	}

	return filepath.Join(filepath.Dir(programPath), name)
}
