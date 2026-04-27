.ONESHELL:

show:
	tmuxinator s talkingheads -p ./talkingheads.yml

actor:
	cd cmd/actor && go build -o ../../build/actor

dialogue:
	cd cmd/dialogue && go build -o ../../build/dialogue

director:
	cd cmd/director && go build -o ../../build/director

deploy-actor:
	cd cmd/actor && GOARCH=arm64 GOOS=linux go build -o ../../build/actor_arm64

flash-action:
	cd action && tinygo flash -target=arduino-uno-q .

mqtt:
	docker run -d --network host eclipse-mosquitto
