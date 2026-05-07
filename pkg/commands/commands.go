package commands

// Direction is the payload for the ask/# MQTT topic.
// It represents a question from the Director to an Actor.
type Direction struct {
	Who  string `json:"who"`
	What string `json:"what"`
}

// Speak is the payload for the speak/# MQTT topic.
// It represents a message spoken by an Actor.
type Speak struct {
	Who  string `json:"who"`
	What string `json:"what"`
}

const (
	StatusSpeaking = "speaking"
	StatusStopped  = "stopped"
)

// Speaking is the payload for the speaking/# MQTT topic.
// It represents a notification from Dialogue when speaking starts or stops.
// Status is either StatusSpeaking or StatusStopped.
type Speaking struct {
	Who    string `json:"who"`
	Status string `json:"status"`
}
