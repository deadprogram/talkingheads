.ONESHELL:

test:
	go test -v $(shell go list ./pkg/... | grep -v hotmic)

show:
	tmuxinator s talkingheads -p ./talkingheads.yml

clean:
	rm -rf build
	mkdir build

actor:
	cd cmd/actor && go build -o ../../build/actor

dialogue:
	cd cmd/dialogue && go build -o ../../build/dialogue

director:
	export WHISPER_DIR=$$(git rev-parse --show-toplevel)/lib/whisper.cpp
	export C_INCLUDE_PATH=$${WHISPER_DIR}/include:$${WHISPER_DIR}/ggml/include
	export LD_LIBRARY_PATH=$${LD_LIBRARY_PATH}:$${WHISPER_DIR}
	export CGO_LDFLAGS="-L$${WHISPER_DIR} -lwhisper -lggml -lm -lstdc++"
	cd cmd/director && go build -o ../../build/director

deploy-actor:
	cd cmd/actor && GOARCH=arm64 GOOS=linux go build -o ../../build/actor_arm64

flash-action:
	cd action && tinygo flash -target=arduino-uno-q .

mqtt:
	docker run -d --network host eclipse-mosquitto

run-dialogue:
	./build/dialogue serve --server localhost:1883 --voice gemmai:en_US:amy-low

run-director:
	./build/director --server localhost:1883

run-gemmai:
	./build/actor --model-path ~/models/Qwen3.5-4B-Uncensored-HauhauCS-Aggressive-Q4_K_M.gguf \
      --server tcp://localhost:1883 \
      --name gemmai \
      --script ./scripts/gemmai.md \
      --script ./scripts/movement.md
