# talking heads

Stop making sense...

## Model server

Start ollama

```shell
docker run --gpus=all -d -v ${HOME}/.ollama:/root/.ollama -p 11434:11434 --name ollama ollama/ollama
```

Stop ollama

```shell
docker stop ollama
```

Subsequent starts

```shell
docker start ollama
```

Download models

```shell
docker exec ollama ollama run llama3
docker exec ollama ollama run phi3
docker exec ollama ollama run gemma
```

## MQTT broker

```shell
docker run --network host eclipse-mosquitto
```

## TTS Engine

https://github.com/rhasspy/piper

- download binary
- add to path
- download voices to `./voices`

## Panelist

```shell
cd cmd
go run ./panelist/ -l="en-US" -voice="hfc_female-medium" -data="../voices" -tts-engine="piper" -model="llama3" -name="llama" -server="localhost:1883"
```

## Moderator

```shell
go run ./moderator/ -server="localhost:1883"
```
