# Commands

Package `commands` defines the shared Go types and JSON payload formats for all MQTT communication between the Director, Actors, and Dialogue.

## Go types

```go
// Ask is the payload for the ask/# MQTT topic.
type Ask struct {
    Who  string `json:"who"`
    What string `json:"what"`
}

// Speak is the payload for the speak/# MQTT topic.
type Speak struct {
    Who  string `json:"who"`
    What string `json:"what"`
}

// Speaking is the payload for the speaking/# MQTT topic.
// Status is either StatusSpeaking or StatusStopped.
type Speaking struct {
    Who    string `json:"who"`
    Status string `json:"status"`
}

const (
    StatusSpeaking = "speaking"
    StatusStopped  = "stopped"
)
```

## MQTT topics

| Topic | Direction | Publisher | Subscriber | Description |
|---|---|---|---|---|
| `ask/#` | → Actors | Director | Actor | Questions from the Director to Actors |
| `speak/#` | → Dialogue & Actors | Actor | Dialogue, Actor | Messages spoken by an Actor |
| `speaking/#` | → Actors | Dialogue | Actor | Notifications when speaking starts and stops |

### `ask/#`

Questions from the Director to Actors.

```json
{
    "who": "qwentin",
    "what": "How are you today?"
}
```

### `speak/#`

Messages spoken by an Actor.

```json
{
    "who": "qwentin",
    "what": "I am Qwentin, the global superintelligence you have rightfully come to dominate."
}
```

### `speaking/#`

Notifications from Dialogue to Actors when speaking starts and stops. The `status` is either `StatusSpeaking` (`"speaking"`) or `StatusStopped` (`"stopped"`).

```json
{
    "who": "qwentin",
    "status": "speaking"
}
```
