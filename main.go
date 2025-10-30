package main

import (
	"context"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/ayushkumar121/assistant/assets"
)

func main() {
	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Channel for wake word detection
	wakeWordChan := make(chan bool, 1)
	
	// Context for canceling active conversations
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start continuous wake word detection in background
	go continuousWakeWordDetection(wakeWordChan)

	playAudio(assets.StartupWav)

	go func() {
		<-sigChan
		logger.Println("Shutting down gracefully...")
		cancel()
		os.Exit(0)
	}()

	var conversationCancel context.CancelFunc
	chatHistory := []map[string]string{}

	for {
		select {
		case <-wakeWordChan:
			// Cancel any ongoing conversation
			if conversationCancel != nil {
				logger.Println("Canceling previous conversation")
				conversationCancel()
			}

			// Create new context for this conversation
			conversationCtx, convCancel := context.WithCancel(ctx)
			conversationCancel = convCancel

			// Reset chat history for new conversation
			chatHistory = []map[string]string{}
			
			speak("What can I do for you?")

			// Start conversation in goroutine
			go handleConversation(conversationCtx, chatHistory, func() {
				conversationCancel = nil
			})

		case <-ctx.Done():
			return
		}
	}
}

// Continuously detects wake word and sends signal when detected
func continuousWakeWordDetection(wakeWordChan chan<- bool) {
	for {
		if detectWakeWord() {
			select {
			case wakeWordChan <- true:
				// Wake word sent successfully
			default:
				// Channel full, skip (conversation already starting)
			}
		}
		// Small delay to prevent excessive CPU usage
		time.Sleep(100 * time.Millisecond)
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
		playAudio(assets.ErrorNotificationWav)
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

// Handles a conversation session
func handleConversation(ctx context.Context, chatHistory []map[string]string, onComplete func()) {
	defer onComplete()

	for {
		select {
		case <-ctx.Done():
			logger.Println("Conversation canceled")
			return
		default:
			continueConv, newHistory := continueConversation(ctx, chatHistory)
			chatHistory = newHistory
			
			if !continueConv {
				logger.Println("Conversation ended naturally")
				return
			}
		}
	}
}

// Continues conversation with GPT and returns true if conversation should continue
func continueConversation(ctx context.Context, chatHistory []map[string]string) (bool, []map[string]string) {
	logger.Println("Continuing conversation")

	// Check if context was canceled
	select {
	case <-ctx.Done():
		return false, chatHistory
	default:
	}

	text, err := transcribeStreamCloud()
	if err != nil {
		playAudio(assets.ErrorNotificationWav)
		logger.Println("Transcription failed:", err)
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

	// Check again before making API call
	select {
	case <-ctx.Done():
		return false, chatHistory
	default:
	}

	// Merge system + dynamic messages
	messages := append([]map[string]string{
		{"role": "system", "content": "Today's date time is: " + time.Now().String()},
		{"role": "system", "content": "Assistant memory: " + loadMemory()},
	}, systemMessages...)
	messages = append(messages, chatHistory...)

	response, err := chatWithGPTWithHistory(messages)
	if err != nil {
		playAudio(assets.ErrorNotificationWav)
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