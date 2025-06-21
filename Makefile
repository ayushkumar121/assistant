LD_FLAGS = -X 'main.OpenAIAPIKey=$(OPENAI_API_KEY)' -X 'main.DebugMode=$(DEBUG_MODE)'

.PHONY: clean

assistant.zip: assistant ffmpeg ffplay assistant_run.sh
	zip assistant.zip assistant ffmpeg ffplay assistant_run.sh
	
assistant: main.go memory.go platform.go config.go
	go build -ldflags="$(LD_FLAGS)" -o assistant .

ffmpeg:
	curl -o ffmpeg.zip -JL https://evermeet.cx/ffmpeg/get/zip
	unzip ffmpeg.zip

ffplay:
	curl -o ffplay.zip -JL https://evermeet.cx/ffmpeg/get/ffplay/zip
	unzip ffplay.zip

clean:
	rm assistant
