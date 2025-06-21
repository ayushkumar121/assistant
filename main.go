package main

import (
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
			"content": "You are a helpful voice assistant. Your pronouns are He/Him. " +
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
		text, err := transcribeStream()
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
		messages = append(messages, map[string]string{
			"role":    "assistant",
			"content": response.Speak,
		})

		// Keep only the last 10 messages again
		if len(messages) > 10 {
			messages = messages[len(messages)-10:]
		}

		saveMemory(response.Memory)
		speak(response.Speak)
	}
}
