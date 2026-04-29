# Talking Heads From The Year 2053

![Talking Heads From The Year 2053 at Gophercon 2024](./images/gophercon-2024-talking-heads.jpg)

A Fantastical Interaction Between Machines From The Future and Their Pet Human

## Architecture

### Overview

```mermaid
flowchart LR
subgraph mqtt broker
    ask
    speak
end
subgraph actors
    actor-1
    actor-2
    actor-3
end
subgraph director
    hotmic
end
subgraph dialogue
    sayanything
end
director -- publish --> ask
ask-- subscribe -->actors
actors-- publish -->speak
speak-- subscribe -->actors
speak-- subscribe -->dialogue
```

### Actor

Actor runs on the Linux part of an Arduino UNO Q board. It is written in Go with [yzma](https://github.com/hybridgroup/yzma) to perform local inference using [llama.cpp](https://github.com/ggml-org/llama.cpp). It communicates with other Actors by publishing and subscribing to [MQTT](https://mqtt.org/) messages.

```mermaid
flowchart LR
subgraph mqtt broker
    ask
    speak
end
subgraph Actor
    subgraph actor
        run
    end
    subgraph tools
        run<-->movement
    end
    subgraph yzma
        run<-->llama.cpp
        llama.cpp-->model
    end
    ask-- subscribe -->run
    run-- publish -->speak
    speak-- subscribe -->run
end
subgraph The Head
    actions<-- UART -->lights
    actions<-- UART -->action
end
```

### The Head

The Head is controlled by the STM32 microcontroller of an Arduino UNO Q board using the `action` firmware written using TinyGo. Actor communicates with The Head using the onboard serial port between the microcontroller and the main processor running on the same Arduino UNO Q board.

```mermaid
flowchart LR
subgraph Arduino UNO Q
    subgraph Microcontroller
        Serial
        GPIO
        UART
    end
    GPIO --> LEDMatrix[LED Matrix]
    subgraph Linux
        Actor<-->Serial
    end
end
subgraph Additional hardware
    GPIO --> WS2812Head[WS2812 Head LEDs]
    UART --> Servo[Feetech Servo]
end
```

### Director

Director runs on a separate computer that is connected to the same local network as the MQTT broker. It uses the Go bindings for [whisper.cpp](https://github.com/ggml-org/whisper.cpp) to perform "push to talk" to communicate with Actors.

```mermaid
flowchart LR
subgraph mqtt broker
    ask
end
subgraph director
    hotmic
end
subgraph hotmic
    whisper.cpp
end
director -- publish --> ask
```

### Dialogue

Dialogue runs on a separate computer that is connected to the same local network as the MQTT broker. It uses the [sayanything](https://github.com/hybridgroup/go-sayanything) package with the [Piper](https://github.com/rhasspy/piper) Text To Speech engine to create audio output for everything said by Actors.

```mermaid
flowchart LR
subgraph mqtt broker
    speak
end
subgraph dialogue
    speak-- subscribe -->sayanything
    subgraph sayanything
        piper-->tts[Text to speech]
    end
    subgraph portaudio
        tts-- WAV -->speaker
    end
end
```

### Detail

```mermaid
flowchart LR
subgraph mqtt broker
    ask
    speak
end
subgraph The Head
    actions<-- UART -->lights
    actions<-- UART -->action
end
subgraph Actor
    subgraph actor
        run
    end
    subgraph tools
        run<-->movement
        movement-->actions
    end
    subgraph yzma
        run<-->llama.cpp
        llama.cpp-->model
    end
    ask-- subscribe -->run
    run-- publish -->speak
    speak-- subscribe -->run
end
subgraph dialogue
    speak-- subscribe -->sayanything
    subgraph sayanything
        piper-->tts[Text to speech]
    end
    subgraph portaudio
        tts-- WAV -->speaker
    end
end
subgraph director
    hotmic
end
subgraph hotmic
    whisper.cpp
end
director -- publish --> ask
```

## Models

More info soon...

## MQTT broker

```shell
docker run -d --network host eclipse-mosquitto
```

## Piper TTS Engine

https://github.com/rhasspy/piper

- download binary
- add to path
- download voices to `./voices`
