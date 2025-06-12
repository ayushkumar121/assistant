package main

var OpenAIAPIKey string

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
