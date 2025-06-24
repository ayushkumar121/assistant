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
	memoryFile         = "memory.txt"
	memoryLimit        = 50
	chatHistoryLimit   = 20
	maxAudioDuration   = 30
	wakeWordDuration   = 3
	maxSilenceDuration = 3
	wakeWord           = "alex"
	whisperURL         = "https://api.openai.com/v1/audio/transcriptions"
	chatURL            = "https://api.openai.com/v1/chat/completions"
	ttsURL             = "https://api.openai.com/v1/audio/speech"

	whisperModel = "whisper-1"
	chatModel    = "gpt-4o-mini"
	ttsModel     = "gpt-4o-mini-tts"
	ttsVoice     = "onyx"
)

var systemMessages = []map[string]string{
	{
		"role": "system",
		"content": "You are a helpful voice assistant. Your pronouns are He/Him. Your name is " + wakeWord +
			"Periodically remind the user of timers, todos and other tasks they have asked you to remember. " +
			"Keep responses short, conversational, and output JSON: " +
			"{\"speak\": \"...\", \"memory\": \"...\", \"continueConversation\": \"true/false\"}. Only respond with valid JSON. " +
			"Only include memory for important information. Return empty string if no important memory is found" +
			"If user asks you to end the conversation, set continueConversation to false" +
			"If user asks you to continue the conversation, set continueConversation to true",
	},
}

var ttsInstructions = `Speak in a friendly, expressive, and natural tone. Use natural pauses and intonation.
 Sound like a real person having a conversation.`

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
