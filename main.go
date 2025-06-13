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

func transcribeStream() (string, error) {
	filePath, err := startAudioCapture()
	if err != nil {
		return "", fmt.Errorf("audio capture error: %v", err)
	}
	defer os.Remove(filePath)

	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open audio file: %v", err)
	}
	defer file.Close()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	part, err := writer.CreateFormFile("file", "audio.wav")
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(part, file); err != nil {
		return "", err
	}
	writer.WriteField("model", whisperModel)
	writer.Close()

	req, err := http.NewRequest("POST", whisperURL, &body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+OpenAIAPIKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

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

type ChatGPTResponse struct {
	Speak  string `json:"speak"`
	Memory string `json:"memory"`
}

func chatWithGPTWithHistory(messages []map[string]string) (*ChatGPTResponse, error) {
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
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+OpenAIAPIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// If status code is not 2xx, decode error
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		errResp, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("OpenAI API error: %s", string(errResp))
	}

	var res struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}

	if len(res.Choices) == 0 {
		return nil, fmt.Errorf("no choices returned from API")
	}
	rawResponse := res.Choices[0].Message.Content

	var response ChatGPTResponse
	if err := json.Unmarshal([]byte(res.Choices[0].Message.Content), &response); err != nil {
		return nil, fmt.Errorf("malformed response: %v", rawResponse)
	}

	return &response, nil
}

func speak(text string) error {
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
	req.Header.Set("Authorization", "Bearer "+OpenAIAPIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return speakFromReader(resp.Body)
}

func main() {
	if OpenAIAPIKey == "" {
		logger.Fatal("❌ OPENAI_API_KEY environment variable not set")
	}

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
