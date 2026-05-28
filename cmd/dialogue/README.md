# dialogue

Text-to-speech service that listens for MQTT messages and speaks them with the [sayanything](https://github.com/talkingheads2053/sayanything) package using the [Piper](https://github.com/rhasspy/piper) Text To Speech engine to create audio output for everything said by Actors.

## Commands

### `serve`

Connect to an MQTT broker and speak any messages published to `speak/#` or `say/#`.

```shell
dialogue serve --server tcp://localhost:1883 \
               --voice llama3000:en_US:joe-medium \
               --voice gemmai:en_US:amy-low
```

| Flag | Alias | Default | Description |
|---|---|---|---|
| `--server` | `-s` | *(required)* | MQTT broker URL |
| `--voice` | `-v` | *(required, repeatable)* | Voice in `name:lang:model` format |
| `--data` | `-d` | `./voices` | Directory containing `.onnx` voice model files |
| `--gpu` | | `false` | Enable GPU acceleration for TTS |

### `say`

Speak a single line and exit — useful for testing a voice without a broker.

```shell
dialogue say --name llama3000 --lang en_US --voice joe-medium --say "Hello world"
```

| Flag | Alias | Default | Description |
|---|---|---|---|
| `--name` | `-n` | *(required)* | Speaker name |
| `--lang` | `-l` | *(required)* | Language code (e.g. `en_US`) |
| `--voice` | `-v` | *(required)* | Voice model name |
| `--data` | `-d` | `./voices` | Directory containing `.onnx` voice model files |
| `--say` | | *(required)* | Text to speak |
| `--gpu` | | `false` | Enable GPU acceleration |

## MQTT message format

Messages must be published to `speak/<name>` as JSON:

```json
{"who": "llama3000", "what": "Hello, I am ready."}
```

The `who` field is matched against the registered voice names. Messages for unknown speakers are silently dropped.

Messages published to `say/<name>` use the same payload shape and are spoken
the same way, but are **not** subscribed to by Actors — use `say/#` to make a
voice speak something without it appearing in any Actor's conversation history.

```json
{"who": "llama3000", "what": "Out-of-band announcement."}
```

When playback starts, Dialogue publishes to `speaking/<name>` (for both `speak/#` and `say/#` messages):

```json
{"who": "llama3000", "status": "speaking"}
```

When playback finishes it publishes:

```json
{"who": "llama3000", "status": "stopped"}
```

All payload types are defined in `pkg/commands`.

## Voice models

Voice model files (`.onnx` + `.onnx.json`) should be placed in the `./voices` directory (or the path passed to `--data`).
