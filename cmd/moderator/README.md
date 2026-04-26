# moderator

Sends questions to panelists over MQTT. Questions can be typed at the keyboard or spoken via a push-to-talk microphone (hotmic mode).

## Flags

| Flag | Required | Default | Description |
|---|---|---|---|
| `--server` | yes | — | MQTT broker address (e.g. `tcp://localhost:1883`) |
| `--hotmic-model` | no | — | Path to a whisper.cpp GGML model file. **Enables hotmic mode when set.** |
| `--hotmic-lang` | no | `auto` | BCP-47 language code for transcription (e.g. `en`), or `auto` for automatic detection |
| `--hotmic-key` | no | ` ` (space) | Keyboard character that toggles recording on/off |

## Keyboard mode (default)

When `--hotmic-model` is not set, questions are read from stdin. Each line must be prefixed with a panelist name followed by a colon or comma:

```
phineas: what is the meaning of life?
llama3000, tell me a joke
```

## Hotmic mode

When `--hotmic-model` is set, the moderator uses the microphone instead of stdin. Press the toggle key once to start recording and again to stop. The audio is transcribed locally using whisper.cpp and parsed using the same `"<panelist>: <question>"` format as keyboard mode.

Press **Ctrl+C** or **Ctrl+D** to exit.

## Building

Hotmic mode requires the whisper.cpp C library. Set these environment variables before building:

```sh
export WHISPER_DIR=$(git rev-parse --show-toplevel)/lib/whisper.cpp
export C_INCLUDE_PATH=$WHISPER_DIR/include:$WHISPER_DIR/ggml/include
export LD_LIBRARY_PATH=$LD_LIBRARY_PATH:$WHISPER_DIR
export CGO_LDFLAGS="-L$WHISPER_DIR -lwhisper -lggml -lm -lstdc++"
```

Then from the `cmd/` directory:

```sh
go build -o moderator ./moderator/...
```

See [pkg/hotmic/README.md](../../pkg/hotmic/README.md) for full instructions on building whisper.cpp from source and downloading a model.

## Example

Keyboard mode:

```sh
./moderator --server tcp://localhost:1883
```

Hotmic mode with the base English model, toggled with the space bar:

```sh
./moderator \
  --server tcp://localhost:1883 \
  --hotmic-model lib/whisper.cpp/models/ggml-base.en.bin \
  --hotmic-lang en \
  --hotmic-key " "
```
