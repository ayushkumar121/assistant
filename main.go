package main

import (
	"strings"
	"time"
)

func main() {
	if OpenAIAPIKey == "" {
		logger.Fatal("❌ OPENAI_API_KEY environment variable not set")
	}

	// Static instructions and memory
	systemMessages := []map[string]string{
		{
			"role": "system",
			"content": "You are a helpful voice assistant. Your pronouns are He/Him. Your name is " + wakeWord +
				"Periodically remind the user of timers, todos and other tasks they have asked you to remember. " +
				"Keep responses short, conversational, and output JSON: " +
				"{\"speak\": \"...\", \"memory\": \"...\"}. Only respond with valid JSON. " +
				"Only include memory for important information. Return empty string if no important memory is found",
		},
		{
			"role":    "system",
			"content": "Assistant memory: " + loadMemory(),
		},
	}

	// This holds only user ↔ assistant turns
	var chatHistory []map[string]string

	for {
		recordingFile, err := recordAudio(recordingDuration)
		if err != nil {
			logger.Println("Transcription failed:", err)
			continue
		}

		text, err := transcribeStreamLocally(recordingFile)
		if err != nil {
			logger.Println("Local Transcription failed:", err)
			continue
		}
		logger.Println("You said:", text)

		if !strings.Contains(strings.ToLower(text), wakeWord) {
			logger.Println("Wake word not detected")
			continue
		}
		logger.Println("Wake word detected:", text)

		text, err = transcribeStreamCloud()
		if err != nil {
			logger.Println("Transcription failed:", err)
			continue
		}
		logger.Println("You said:", text)

		chatHistory = append(chatHistory, map[string]string{
			"role":    "user",
			"content": text,
		})

		// Trim to last `chatHistoryLimit` non-system messages
		if len(chatHistory) > chatHistoryLimit {
			chatHistory = chatHistory[len(chatHistory)-chatHistoryLimit:]
		}

		// Merge system + dynamic messages
		messages := append([]map[string]string{
			{"role": "system", "content": "Today's date time is: " + time.Now().String()},
		}, systemMessages...)
		messages = append(messages, chatHistory...)

		// Send to GPT with context
		response, err := chatWithGPTWithHistory(messages)
		if err != nil {
			logger.Println("ChatGPT error:", err)
			return
		}
		logger.Println("GPT says:", response.Speak)
		logger.Println("GPT will remember:", response.Memory)

		// Add assistant message
		chatHistory = append(chatHistory, map[string]string{
			"role":    "assistant",
			"content": response.Speak,
		})

		saveMemory(response.Memory)
		speak(response.Speak)
	}
}
