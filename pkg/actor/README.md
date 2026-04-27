# pkg/actor

Package `actor` manages the conversation loop with the language model and exposes tool calls that control the actor's physical head 
movement.

## Key types

### `Actor`

Created with `NewActor`. Runs a conversation loop against the configured model, dispatching any tool calls the model emits to the registered tools.

```go
a, err := actor.NewActor(mp, commander, moreFunc, outputFunc)
```

- `mp` — path to the model file(s)
- `commander` — where action commands are sent (`nil` logs to console)
- `moreFunc` — called each turn to append new user input to the conversation
- `outputFunc` — called with each text response from the model

### `Commander`

Interface used to send action commands to the microcontroller.

| Implementation | Description |
|---|---|
| `LogCommander` (default) | Logs commands via `log.Printf` |
| `SerialCommander` | Sends commands over a serial port |

```go
// Serial
sc, err := actor.NewSerialCommander("/dev/ttyACM0", 9600)

// Console (nil also works)
var commander actor.Commander // nil → LogCommander
```

### `Movement` tool

Registered automatically by `NewActor`. Exposes the following commands to the model:

| Command | Angle required | Description |
|---|---|---|
| `look` | yes (0–180) | Turn head to angle immediately |
| `slowlook` | yes (0–180) | Turn head to angle slowly |
| `headshake` | no | Shake head (indicates "no") |
| `wait` | no | Idle movement while waiting to speak |
| `speak` | no | Small movement while speaking |
| `stop` | no | Stop and center (90°) |

Angle convention: `0` = full right, `90` = center, `180` = full left.

## Serial protocol

Commands are sent as plain ASCII terminated with `\r`:

```
look 135\r
headshake\r
stop\r
```

This matches the serial API implemented by the `action` firmware in `action/`.
