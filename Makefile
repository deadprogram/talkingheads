.ONESHELL:

practice:
	tmuxinator s talkingheads -p ./talkingheads-noserial.yml

show:
	tmuxinator s talkingheads -p ./talkingheads-serial.yml

actor:
	cd cmd/actor && go build -o ../../build/actor

dialogue:
	cd cmd/dialogue && go build -o ../../build/dialogue

