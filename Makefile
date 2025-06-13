LD_FLAGS = -X 'main.OpenAIAPIKey=$(OPENAI_API_KEY)' -X 'main.DebugMode=$(DEBUG)'

assistant: main.go memory.go platform.go config.go
	go build -ldflags="$(LD_FLAGS)" -o assistant .
