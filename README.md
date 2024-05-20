# talking head

Stop making sense...


## Running

Start ollama

```shell
docker run -d -v ollama:/home/ron/.ollama -p 11434:11434 --name ollama ollama/ollama
```

Run the CLI

```shell
cd cmd
go run . -k /home/ron/sayanything-383222-88419296b765.json -l="en-US" -voice="en-US-Neural2-D"
```
