package main

import (
	"io"
	"log"
	"os"
)

var (
	OpenAIAPIKey string
	DebugMode    string
)

const (
	memoryFile       = "memory.txt"
	memoryLimit      = 50
	chatHistoryLimit = 20
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

func init() {
	if DebugMode != "" {
		debugLogger.SetOutput(os.Stderr)
	}
}
