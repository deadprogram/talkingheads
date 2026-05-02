.ONESHELL:

test: test-pkg test-action

test-pkg:
	go test -v $(shell go list ./pkg/... | grep -v hotmic)

test-action:
	cd action && go test -v

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

arduino-actor:
	cd cmd/actor && GOARCH=arm64 GOOS=linux go build -o ../../build/actor_arm64

flash-arduino-actor:
	# TODO: stop the running service on the Arduino before flashing
	adb push ./build/actor_arm64 /home/arduino/actor

flash-action:
	cd action && tinygo flash -target=arduino-uno-q -tags noservo .

mqtt:
	docker run -d --network host eclipse-mosquitto

show:
	tmuxinator s talkingheads -p ./talkingheads.yml
