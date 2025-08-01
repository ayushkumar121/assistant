package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

func transcribeStreamLocally(recordingFileName string) (string, error) {
	cmd := exec.Command(resolveExecutablePath("whisper.cpp-1.7.5/build/bin/whisper-cli"),
		"-m", resolveExecutablePath("whisper.cpp-1.7.5/models/ggml-tiny.en.bin"),
		"-nt", "-np",
		"-f", recordingFileName,
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}

	if err := cmd.Start(); err != nil {
		return "", err
	}

	buf, err := io.ReadAll(stdout)
	if err != nil {
		return "", err
	}

	if err := cmd.Wait(); err != nil {
		return "", err
	}

	return strings.TrimSpace(string(buf)), nil
}

func transcribeStreamCloud() (string, error) {
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
	writer.WriteField("language", "en")
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
	Speak                string `json:"speak"`
	Memory               string `json:"memory"`
	ContinueConversation bool   `json:"continueConversation"`
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
						"speak":                map[string]any{"type": "string"},
						"memory":               map[string]any{"type": "string"},
						"continueConversation": map[string]any{"type": "boolean"},
					},
					"required":             []string{"speak", "memory", "continueConversation"},
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
		"model":           ttsModel,
		"input":           text,
		"instructions":    ttsInstructions,
		"voice":           ttsVoice,
		"response_format": "wav",
		"sample_rate":     16000,
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

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		errResp, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("OpenAI API error: %s", string(errResp))
	}

	return speakFromReader(resp.Body)
}
