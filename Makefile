LD_FLAGS = -X 'main.OpenAIAPIKey=$(OPENAI_API_KEY)' -X 'main.DebugMode=$(DEBUG_MODE)'
WHISPER_VERSION=1.7.5
WHISPER_RELEASE_URL=https://github.com/ggml-org/whisper.cpp/archive/refs/tags/v$(WHISPER_VERSION).zip
WHISPER_DIR=whisper.cpp-$(WHISPER_VERSION)
WHISPER_BINARY = $(WHISPER_DIR)/build/bin/whisper-cli

.PHONY: clean

assistant: main.go memory.go platform.go openai.go config.go $(WHISPER_BINARY)
	go build -ldflags="$(LD_FLAGS)" -o assistant .

$(WHISPER_BINARY): whisper.zip
	unzip -o whisper.zip
	cd $(WHISPER_DIR) && make && bash ./models/download-ggml-model.sh tiny.en

whisper.zip:
	wget -O whisper.zip https://github.com/ggml-org/whisper.cpp/archive/refs/tags/v1.7.5.zip

assistant.zip: assistant ffmpeg ffplay assistant_run.sh
	zip -r assistant.zip assistant ffmpeg ffplay assistant_run.sh $(WHISPER_DIR)/build $(WHISPER_DIR)/models

ffmpeg:
	curl -o ffmpeg.zip -JL https://evermeet.cx/ffmpeg/get/zip
	unzip ffmpeg.zip

ffplay:
	curl -o ffplay.zip -JL https://evermeet.cx/ffmpeg/get/ffplay/zip
	unzip ffplay.zip

clean:
	rm assistant
	rm -rf $(WHISPER_DIR)/build
