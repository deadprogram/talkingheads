# director

Sends questions to panelists over MQTT. Questions can be typed at the keyboard or spoken via a push-to-talk microphone (hotmic mode).

## Flags

| Flag | Required | Default | Description |
|---|---|---|---|
| `--server` | yes | — | MQTT broker address (e.g. `tcp://localhost:1883`) |
| `--actor` | yes | — | Canonical actor name (repeatable; e.g. `--actor gemmai --actor phineas`) |
| `--hotmic-model` | no | — | Path to a whisper.cpp GGML model file. **Enables hotmic mode when set.** |
| `--hotmic-lang` | no | `auto` | BCP-47 language code for transcription (e.g. `en`), or `auto` for automatic detection |
| `--hotmic-key` | no | ` ` (space) | Keyboard character that toggles recording on/off |
| `--hotmic-fuzzy-threshold` | no | `0.6` | Maximum Levenshtein distance ratio (0–1) for fuzzy actor name matching; lower values require a closer match |
| `--hotmic-actor-alias` | no | — | Map alternate spoken names to a canonical actor: `--hotmic-actor-alias "gemmai:jami\|jamai\|jenna"` (repeatable) |

## MQTT message format

Questions are published to `ask/<name>` as JSON:

```json
{"who": "llama3000", "what": "What is the meaning of life?"}
```

The `who` field is the panelist name (lower-cased from the prefix). Payload types are defined in `pkg/commands`.

## Keyboard mode (default)

When `--hotmic-model` is not set, questions are read from stdin. Each line must be prefixed with an actor name followed by a colon or comma:

```
phineas: what is the meaning of life?
llama3000, tell me a joke
```

## Hotmic mode

When `--hotmic-model` is set, the moderator uses the microphone instead of stdin. Press the toggle key once to start recording and again to stop. The audio is transcribed locally using whisper.cpp and parsed using the same `"<actor>: <question>"` format as keyboard mode.

The spoken actor name is lowercased and matched against the configured actors using three strategies in order:

1. **Exact match** after normalisation (lowercase, alphanumeric only).
2. **Substring containment** — handles e.g. "lama" → `llama3000`.
3. **Fuzzy edit-distance** — accepts the closest actor when the Levenshtein distance is within `--hotmic-fuzzy-threshold` (default `0.6`) of the longer name. Lower the threshold to require a closer match.

Before all three strategies, **aliases** defined with `--hotmic-actor-alias` are checked first. Use this to handle systematic whisper.cpp mis-transcriptions:

```sh
--hotmic-actor-alias "gemmai:jami|jamai|jenna|jedi"
```

Press **Ctrl+C** or **Ctrl+D** to exit.

## Building

Hotmic mode requires the whisper.cpp C library. Set these environment variables before building:

```sh
export WHISPER_DIR=$(git rev-parse --show-toplevel)/lib/whisper.cpp
export C_INCLUDE_PATH=$WHISPER_DIR/include:$WHISPER_DIR/ggml/include
export LD_LIBRARY_PATH=$LD_LIBRARY_PATH:$WHISPER_DIR
export CGO_LDFLAGS="-L$WHISPER_DIR -lwhisper -lggml -lm -lstdc++"
```

Then from the `cmd/director` directory:

```sh
go build -o ../../build/director .
```

See [pkg/hotmic/README.md](../../pkg/hotmic/README.md) for full instructions on building whisper.cpp from source and downloading a model.

## Example

Keyboard mode:

```sh
./director --server tcp://localhost:1883 \
  --actor llama3000 --actor phineas --actor gemmai --actor qwentin
```

Hotmic mode with the base English model, toggled with the space bar:

```sh
./director \
  --server tcp://localhost:1883 \
  --actor llama3000 --actor phineas --actor gemmai --actor qwentin \
  --hotmic-model lib/whisper.cpp/models/ggml-base.en.bin \
  --hotmic-lang en \
  --hotmic-key " " \
  --hotmic-fuzzy-threshold 0.4 \
  --hotmic-actor-alias "gemmai:jami|jamai|jenna|jedi"
```
