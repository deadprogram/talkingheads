# actor

Command-line program that runs Actor in a conversation loop.

## Usage

```
actor [options]
```

One of `--model-url` or `--model-path` is required.

## Flags

| Flag | Alias | Default | Description |
|---|---|---|---|
| `--model-url` | `-u` | | HuggingFace or other URL to download the model from |
| `--model-path` | `-p` | | Path to a pre-downloaded model file |
| `--system-prompt` | `-s` | `"You are a helpful assistant."` | System prompt that sets the actor's persona |
| `--server` | `-b` | | MQTT broker URL (e.g. `tcp://localhost:1883`); enables MQTT mode |
| `--name` | `-n` | `actor` | Actor name used for MQTT topics `ask/<name>` and `speak/<name>` |
| `--serial` | | | Serial port for sending action commands to the microcontroller (e.g. `/dev/ttyACM0`) |
| `--baud` | | `9600` | Baud rate for the serial port |

## Modes

**Interactive (default)** — reads input from stdin, prints responses to stdout.

**MQTT** — set `--server` to connect to an MQTT broker. The actor subscribes to
`ask/<name>` for direct prompts and `speak/#` to hear other actors, and publishes
its responses to `speak/<name>`.

**Serial** — set `--serial` to send head movement commands to a microcontroller
running the action firmware. If omitted, commands are logged to the console instead.

## Examples

```shell
# Interactive, local model
actor --model-path ./models/llama.gguf

# MQTT mode with a custom persona
actor --model-path ./models/llama.gguf \
      --server tcp://localhost:1883 \
      --name bob \
      --system-prompt "You are a pirate named Bob."

./build/actor --model-path gemma-3-1b-it-heretic-extreme-uncensored-abliterated.i1-Q4_K_M.gguf --server tcp://localhost:1883 --name gemmai --system-prompt "You are a pirate named Gemmai."

# With serial head control
actor --model-path ./models/llama.gguf --serial /dev/ttyACM0 --baud 9600
```
