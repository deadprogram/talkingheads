# pkg/hotmic

Push-to-talk microphone capture with local speech-to-text transcription.

Press a configurable key once to start recording and again to stop. The captured audio is transcribed offline using a [whisper.cpp](https://github.com/ggml-org/whisper.cpp) model via [ardanlabs/bucky](https://github.com/ardanlabs/bucky) and returned as a plain string.

## Dependencies

| Dependency | Purpose |
|---|---|
| [gordonklaus/portaudio](https://github.com/gordonklaus/portaudio) | Audio capture via PortAudio |
| [ardanlabs/bucky](https://github.com/ardanlabs/bucky) | Speech-to-text via whisper.cpp (purego FFI, no CGo) |
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

### 2. libwhisper shared library

Bucky loads `libwhisper.so` at runtime via `dlopen`. You need a pre-built shared library; no CGo or static library is required.

Build with CPU-only support:

```sh
git clone https://github.com/ggml-org/whisper.cpp
cd whisper.cpp
cmake -B build -DBUILD_SHARED_LIBS=ON -DGGML_OPENMP=OFF -DWHISPER_BUILD_TESTS=OFF -DWHISPER_BUILD_EXAMPLES=OFF
cmake --build build --config Release -j$(nproc)
# Copy the resulting shared library to a stable location, e.g. ~/Development/bucky/lib/
cp build/src/libwhisper.so ~/Development/bucky/lib/
```

At runtime, point `BUCKY_LIB` at the directory containing `libwhisper.so`:

```sh
export BUCKY_LIB=~/Development/bucky/lib
```

### 3. A whisper model file

Download any GGML-format model, for example:

```sh
# Using the whisper.cpp helper script:
bash whisper.cpp/models/download-ggml-model.sh base.en
# Or download directly from Hugging Face:
wget https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-base.en.bin
```

## Building

Bucky uses purego FFI — no CGo build flags are required. Build normally:

```sh
go build ./pkg/hotmic/...
```

Or build the whole project:

```sh
go build ./...
```

## Usage

```go
import "github.com/talkingheads2053/talkingheads/pkg/hotmic"

mic, err := hotmic.New(hotmic.Options{
    Key:       ' ',                    // space toggles record on/off
    LibPath:   "/path/to/bucky/lib",   // dir containing libwhisper.so; defaults to $BUCKY_LIB
    ModelPath: "/path/to/ggml-base.en.bin",
    Language:  "en",                   // or "auto"
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
