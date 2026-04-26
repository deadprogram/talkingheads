# pkg/hotmic

Push-to-talk microphone capture with local speech-to-text transcription.

Press a configurable key once to start recording and again to stop. The captured audio is transcribed offline using a [whisper.cpp](https://github.com/ggml-org/whisper.cpp) model and returned as a plain string.

## Dependencies

| Dependency | Purpose |
|---|---|
| [gordonklaus/portaudio](https://github.com/gordonklaus/portaudio) | Audio capture via PortAudio |
| [ggerganov/whisper.cpp bindings/go](https://github.com/ggml-org/whisper.cpp/tree/master/bindings/go) | Speech-to-text (CGo, local build required) |
| [golang.org/x/term](https://pkg.go.dev/golang.org/x/term) | Raw terminal key detection |

## Prerequisites

### 1. PortAudio

Install the PortAudio development headers:

```sh
# Debian / Ubuntu
sudo apt install portaudio19-dev

# Fedora
sudo dnf install portaudio-devel

# macOS (Homebrew)
brew install portaudio
```

### 2. whisper.cpp (built from source)

The Go module uses a `replace` directive pointing at a local whisper.cpp checkout because the published module does not bundle the required C libraries. You need to build `libwhisper.a` yourself:

```sh
git clone https://github.com/ggml-org/whisper.cpp.git ~/Development/whisper.cpp
cd ~/Development/whisper.cpp
cmake -B build -DBUILD_SHARED_LIBS=OFF
cmake --build build --config Release -j$(nproc)
# The resulting static libraries are written to the repo root:
cp build/src/libwhisper.a .
cp build/ggml/src/libggml.a .
```

> **Note:** The `go.mod` in this repository has a `replace` directive that maps
> `github.com/ggerganov/whisper.cpp/bindings/go` to the local checkout:
> ```
> replace github.com/ggerganov/whisper.cpp/bindings/go => /home/ron/Development/whisper.cpp/bindings/go
> ```
> Adjust this path if your clone is in a different location.

### 3. A whisper model file

Download any GGML-format model, for example:

```sh
cd ~/Development/whisper.cpp
bash models/download-ggml-model.sh base.en
# Model is saved to models/ggml-base.en.bin
```

## Building

Because the whisper.cpp bindings use CGo, you must tell the compiler where to find the headers and the static library. Set the following environment variables before `go build` (or export them in your shell profile):

```sh
export C_INCLUDE_PATH=/home/ron/Development/whisper.cpp/include:/home/ron/Development/whisper.cpp/ggml/include
export LIBRARY_PATH=/home/ron/Development/whisper.cpp
```

Then build normally:

```sh
CGO_LDFLAGS="-L/home/ron/Development/whisper.cpp -lwhisper -lggml -lm -lstdc++" \
go build ./pkg/hotmic/...
```

Or build the whole project:

```sh
CGO_LDFLAGS="-L/home/ron/Development/whisper.cpp -lwhisper -lggml -lm -lstdc++" \
go build ./...
```

### One-liner

```sh
C_INCLUDE_PATH=/home/ron/Development/whisper.cpp/include:/home/ron/Development/whisper.cpp/ggml/include \
LIBRARY_PATH=/home/ron/Development/whisper.cpp \
CGO_LDFLAGS="-L/home/ron/Development/whisper.cpp -lwhisper -lggml -lm -lstdc++" \
go build ./pkg/hotmic/...
```

## Usage

```go
import "github.com/deadprogram/talkingheads/pkg/hotmic"

mic, err := hotmic.New(hotmic.Options{
    Key:       ' ',                                          // space toggles record on/off
    ModelPath: "/home/ron/Development/whisper.cpp/models/ggml-base.en.bin",
    Language:  "en",                                        // or "auto"
})
if err != nil {
    log.Fatal(err)
}
defer mic.Close()

ctx, cancel := context.WithCancel(context.Background())
defer cancel()

texts, err := mic.Listen(ctx)
if err != nil {
    log.Fatal(err)
}

for text := range texts {
    fmt.Println(text)
}
```

Press the toggle key once to begin recording and again to stop. The transcribed text is sent on the returned channel. Press **Ctrl+C** or **Ctrl+D** to exit.
