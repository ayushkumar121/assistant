package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/ayushkumar121/assistant/assets"
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

	speak("Hi, I'm Alex", nil)
	for {
		if !conversationActive {
			if detectWakeWord() {
				conversationActive = true
				speak("What can I do for you?", nil)
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
	var err error
	defer func() {
		if err != nil {
			debugLogger.Println(err)
			playAudio(assets.ErrorNotificationWav)
		}
	}()

	logger.Println("Continuing conversation")
	text, err := transcribeStreamCloud()
	if err != nil {
		err = fmt.Errorf("transcription error: %v", err)
		return false, chatHistory
	}
	playAudio(assets.NotificationWav)
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

	response, err := chatResponse(messages)
	if err != nil {
		err = fmt.Errorf("gpt error: %v", err)
		return false, chatHistory
	}
	logger.Println("GPT says:", response.Speak)
	logger.Println("GPT will remember:", response.Memory)

	saveMemory(response.Memory)

	err = speakWithInterrupt(response.Speak)
	if err != nil {
		err = fmt.Errorf("speach error: %v", err)
		return false, chatHistory
	}

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

func speakWithInterrupt(text string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	interrupt := make(chan struct{}, 1)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				detected, err := listenForVoiceActivity(interruptDetectDuration)
				if err != nil {
					debugLogger.Println(err)
					continue
				}

				if detected {
					select {
					case interrupt <- struct{}{}:
					default:
					}
					return
				}
			}
		}
	}()
	return speak(text, interrupt)
}
