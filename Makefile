assistant: main.go memory.go platform.go config.go
	go build -ldflags="-X 'main.OpenAIAPIKey=$$OPENAI_API_KEY'" -o assistant .
