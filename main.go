package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"time"
)

const (
	memoryFile       = "memory.txt"
	memoryLimit      = 50
	chatHistoryLimit = 20
	whisperURL       = "https://api.openai.com/v1/audio/transcriptions"
	chatURL          = "https://api.openai.com/v1/chat/completions"
	ttsURL           = "https://api.openai.com/v1/audio/speech"

	chatModel = "gpt-4.1"
	ttsModel  = "tts-1"
	ttsVoice  = "alloy"
)

func getAPIKey() string {
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		fmt.Println("❌ OPENAI_API_KEY environment variable not set")
		os.Exit(1)
	}
	return key
}

func transcribeStream(apiKey string, duration int) (string, error) {
	audioStream, err := startAudioCapture(duration)
	if err != nil {
		return "", fmt.Errorf("audio stream error: %v", err)
	}
	defer audioStream.Close()

	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)

	// stream audio + form data into pipe
	go func() {
		defer pw.Close()

		part, _ := writer.CreateFormFile("file", "audio.wav")
		io.Copy(part, audioStream)

		writer.WriteField("model", "whisper-1")
		writer.Close()
	}()

	req, err := http.NewRequest("POST", whisperURL, pr)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// If status code is not 2xx, decode error
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		errResp, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("OpenAI API error: %s", string(errResp))
	}

	var res struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", err
	}

	return res.Text, nil
}

func chatWithGPTWithHistory(apiKey string, messages []map[string]string) (string, error) {
	bodyData := map[string]any{
		"model":    chatModel,
		"messages": messages,
		"response_format": map[string]any{
			"type": "json_schema",
			"json_schema": map[string]any{
				"name": "assistant_response",
				"schema": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"speak":  map[string]any{"type": "string"},
						"memory": map[string]any{"type": "string"},
					},
					"required":             []string{"speak", "memory"},
					"additionalProperties": false,
				},
				"strict": true,
			},
		},
	}
	body, _ := json.Marshal(bodyData)

	req, err := http.NewRequest("POST", chatURL, bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// If status code is not 2xx, decode error
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		errResp, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("OpenAI API error: %s", string(errResp))
	}

	var res struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", err
	}

	if len(res.Choices) == 0 {
		return "", fmt.Errorf("no choices returned from API")
	}

	return res.Choices[0].Message.Content, nil
}

func speak(apiKey, text string) error {
	bodyData := map[string]any{
		"model": ttsModel,
		"input": text,
		"voice": ttsVoice,
	}
	body, _ := json.Marshal(bodyData)

	req, err := http.NewRequest("POST", ttsURL, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return speakFromReader(resp.Body)
}

func main() {
	apiKey := getAPIKey()

	// Static instructions and memory
	systemMessages := []map[string]string{
		{
			"role": "system",
			"content": "You are a helpful voice assistant. " +
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
		text, err := transcribeStream(apiKey, 10)
		if err != nil {
			fmt.Println("Transcription failed:", err)
			return
		}
		fmt.Println("🗣️ You said:", text)

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
		reply, err := chatWithGPTWithHistory(apiKey, messages)
		if err != nil {
			fmt.Println("ChatGPT error:", err)
			return
		}
		fmt.Println("🤖 GPT says:", reply)

		// Add assistant message
		messages = append(messages, map[string]string{
			"role":    "assistant",
			"content": reply,
		})

		// Keep only the last 10 messages again
		if len(messages) > 10 {
			messages = messages[len(messages)-10:]
		}

		var result struct {
			Speak  string `json:"speak"`
			Memory string `json:"memory"`
		}

		if err := json.Unmarshal([]byte(reply), &result); err != nil {
			// fallback: speak full response
			speak(apiKey, reply)
		} else {
			saveMemory(result.Memory)
			speak(apiKey, result.Speak)
		}
	}
}
