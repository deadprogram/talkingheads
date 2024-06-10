# talking heads

Stop making sense...


## Running

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
docker exec ollama ollama run llama2
```



Run the CLI

```shell
cd cmd
go run ./panelist -k /home/ron/sayanything-383222-88419296b765.json -l="en-US" -voice="en-US-Neural2-D"
```


Running MQTT broker

```shell
docker run --network host eclipse-mosquitto
```
