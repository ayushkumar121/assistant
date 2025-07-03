package main

import (
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

func main() {
	if OpenAIAPIKey == "" {
		logger.Fatal("❌ OPENAI_API_KEY environment variable not set")
	}

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		logger.Println("Shutting down gracefully...")
		os.Exit(0)
	}()

	// This holds only user ↔ assistant turns
	chatHistory := []map[string]string{}
	conversationActive := false

	speak("Hi, I'm Alex")
	for {
		if !conversationActive {
			if detectWakeWord() {
				conversationActive = true
				speak("What can I do for you?")
			}
		} else {
			// Start conversation
			conversationActive, chatHistory = continueConversation(chatHistory)
			if !conversationActive {
				chatHistory = []map[string]string{}
			}
		}
	}
}

// Detects wake word and returns true if detected
func detectWakeWord() bool {
	logger.Println("Detecting wake word")
	recordingFile, err := recordAudio(wakeWordDuration)
	if err != nil {
		logger.Println("Transcription failed:", err)
		return false
	}

	text, err := transcribeStreamLocally(recordingFile)
	if err != nil {
		logger.Println("No valid speech detected")
		return false
	}
	logger.Println("Transcription:", text)

	if !strings.Contains(strings.ToLower(text), wakeWord) {
		logger.Println("Wake word not detected")
		return false
	}
	logger.Println("Wake word detected")
	return true
}

// Continues conversation with GPT and returns true if conversation should continue
func continueConversation(chatHistory []map[string]string) (bool, []map[string]string) {
	logger.Println("Continuing conversation")
	text, err := transcribeStreamCloud()
	if err != nil {
		logger.Println("Transcription failed:", err)
		return false, chatHistory
	}
	logger.Println("You said:", text)

	chatHistory = append(chatHistory, map[string]string{
		"role":    "user",
		"content": text,
	})

	// Trim to last chatHistoryLimit non-system messages
	if len(chatHistory) > chatHistoryLimit {
		chatHistory = chatHistory[len(chatHistory)-chatHistoryLimit:]
	}

	// Merge system + dynamic messages
	messages := append([]map[string]string{
		{"role": "system", "content": "Today's date time is: " + time.Now().String()},
		{"role": "system", "content": "Assistant memory: " + loadMemory()},
	}, systemMessages...)
	messages = append(messages, chatHistory...)

	response, err := chatWithGPTWithHistory(messages)
	if err != nil {
		logger.Println("ChatGPT error:", err)
		return false, chatHistory
	}
	logger.Println("GPT says:", response.Speak)
	logger.Println("GPT will remember:", response.Memory)

	saveMemory(response.Memory)
	speak(response.Speak)

	chatHistory = append(chatHistory, map[string]string{
		"role":    "assistant",
		"content": response.Speak,
	})

	if !response.ContinueConversation {
		logger.Println("Conversation ended")
		return false, chatHistory
	}

	return true, chatHistory
}
