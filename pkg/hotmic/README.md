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
git submodule update --init lib/whisper.cpp
cd lib/whisper.cpp
cmake -B build -DBUILD_SHARED_LIBS=OFF -DGGML_OPENMP=OFF -DWHISPER_BUILD_TESTS=OFF -DWHISPER_BUILD_EXAMPLES=OFF
cmake --build build --config Release -j$(nproc)
# The resulting static libraries are written to the repo root:
cp build/src/libwhisper.a .
cp build/ggml/src/libggml.a .
```

> **Note:** The `go.mod` files in this repository have a `replace` directive that maps
> `github.com/ggerganov/whisper.cpp/bindings/go` to the submodule:
> ```
> replace github.com/ggerganov/whisper.cpp/bindings/go => ./lib/whisper.cpp/bindings/go
> ```
> No manual adjustment is needed as long as you initialised the submodule.

### 3. A whisper model file

Download any GGML-format model, for example:

```sh
cd lib/whisper.cpp
bash models/download-ggml-model.sh base.en
# Model is saved to lib/whisper.cpp/models/ggml-base.en.bin
```

## Building

Because the whisper.cpp bindings use CGo, you must tell the compiler where to find the headers and the static library. Set the following environment variables before `go build` (or export them in your shell profile):

```sh
export WHISPER_DIR=$(git rev-parse --show-toplevel)/lib/whisper.cpp
export C_INCLUDE_PATH=$WHISPER_DIR/include:$WHISPER_DIR/ggml/include
export LIBRARY_PATH=$WHISPER_DIR
```

Then build normally:

```sh
CGO_LDFLAGS="-L$WHISPER_DIR -lwhisper -lggml -lm -lstdc++" \
go build ./pkg/hotmic/...
```

Or build the whole project:

```sh
CGO_LDFLAGS="-L$WHISPER_DIR -lwhisper -lggml -lm -lstdc++" \
go build ./...
```

### One-liner

```sh
WHISPER_DIR=$(git rev-parse --show-toplevel)/lib/whisper.cpp \
C_INCLUDE_PATH=$WHISPER_DIR/include:$WHISPER_DIR/ggml/include \
LIBRARY_PATH=$WHISPER_DIR \
CGO_LDFLAGS="-L$WHISPER_DIR -lwhisper -lggml -lm -lstdc++" \
go build ./pkg/hotmic/...
```

## Usage

```go
import "github.com/deadprogram/talkingheads/pkg/hotmic"

mic, err := hotmic.New(hotmic.Options{
    Key:       ' ',                                          // space toggles record on/off
    ModelPath: "lib/whisper.cpp/models/ggml-base.en.bin",
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
