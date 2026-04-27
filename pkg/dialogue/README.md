# pkg/dialogue

Package `dialogue` uses the [sayanything](https://github.com/hybridgroup/go-sayanything) package with the [Piper](https://github.com/rhasspy/piper) Text To Speech engine to create audio output for everything said by Actors.

## Key types

### `Voice`

Wraps a Piper TTS engine and audio player for a named speaker.

```go
v, err := dialogue.NewVoice(name, lang, voiceModel, dataDir, gpu)
```

- `name` — speaker identifier matched against the `who` field in MQTT messages
- `lang` — language code (e.g. `en_US`)
- `voiceModel` — voice model name (e.g. `en_US-joe-medium`)
- `dataDir` — path to the directory containing `.onnx` model files
- `gpu` — enable GPU acceleration

| Method | Description |
|---|---|
| `SayOnce(text)` | Speak synchronously, blocking until playback finishes |
| `SayAnything(text)` | Speak asynchronously in a goroutine |

### `Listener`

Subscribes to `speak/#` on an MQTT broker and routes each message to the matching `Voice`.

```go
listener, err := dialogue.NewListener(clientID, brokerURL, voices)
listener.Listen() // blocks
```

`Listen` blocks, consuming messages from the internal channel. Each incoming message is decoded as `SomethingSaid` and dispatched to the voice whose `Name` matches the `who` field.

### `SomethingSaid`

The JSON payload expected on `speak/#` topics:

```go
type SomethingSaid struct {
    Who  string `json:"who"`
    What string `json:"what"`
}
```

## MQTT topics

| Topic | Direction | Description |
|---|---|---|
| `speak/#` | subscribe | Messages to be spoken; routed by `who` field |
