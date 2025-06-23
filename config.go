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
	memoryFile        = "memory.txt"
	memoryLimit       = 50
	chatHistoryLimit  = 20
	maxAudioDuration  = 30
	recordingDuration = 3
	wakeWord          = "alex"
	whisperURL        = "https://api.openai.com/v1/audio/transcriptions"
	chatURL           = "https://api.openai.com/v1/chat/completions"
	ttsURL            = "https://api.openai.com/v1/audio/speech"

	whisperModel = "whisper-1"
	chatModel    = "gpt-4o-mini"
	ttsModel     = "gpt-4o-mini-tts"
	ttsVoice     = "onyx"
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
