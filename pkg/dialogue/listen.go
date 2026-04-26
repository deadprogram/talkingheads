package dialogue

import (
	"encoding/json"
	"errors"
	"log"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// SomethingSaid represents a message that was said, including who said it and what they said.
type SomethingSaid struct {
	Who  string `json:"who"`
	What string `json:"what"`
}

// Listener listens for MQTT messages with something that was said, and then tells the appropriate Voice to say it.
type Listener struct {
	whatWasSaid chan SomethingSaid
	voices      map[string]*Voice
}

// NewListener starts the MQTT client and subscribes to the topic for something that was said.
// It returns a Listener that can be used to listen for messages and tell the appropriate Voice to say it.
func NewListener(name, server string, voices map[string]*Voice) (*Listener, error) {
	if len(voices) == 0 {
		return nil, errors.New("at least one voice is required")
	}

	options := mqtt.NewClientOptions()
	options.AddBroker(server)
	options.SetClientID(name)
	options.KeepAlive = 300

	log.Printf("Connecting to MQTT broker at %s\n", server)
	client := mqtt.NewClient(options)
	token := client.Connect()
	if token.Wait() && token.Error() != nil {
		log.Fatal("failed creating MQTT client: ", token.Error())
		return nil, token.Error()
	}

	m := &Listener{
		whatWasSaid: make(chan SomethingSaid, 5),
		voices:      voices,
	}

	speakTopic := "speak/#"
	token = client.Subscribe(speakTopic, 0, m.handleSpeaking)
	if token.Wait() && token.Error() != nil {
		log.Fatal("failed subscribing to MQTT topic: ", token.Error())
		return nil, token.Error()
	}
	log.Printf("Subscribed to topic %s\n", speakTopic)

	return m, nil
}

func (m *Listener) handleSpeaking(client mqtt.Client, msg mqtt.Message) {
	log.Printf("Received message: %s\n", string(msg.Payload()))

	// parse the payload and extract the text to speak
	var s SomethingSaid
	if err := json.Unmarshal(msg.Payload(), &s); err != nil {
		log.Printf("Failed to unmarshal message: %v\n", err)
		return
	}
	m.whatWasSaid <- s
}

// Listen listens for messages on the whatWasSaid channel and tells the appropriate Voice to say it.
func (m *Listener) Listen() {
	for s := range m.whatWasSaid {
		log.Printf("Received something said: %s\n", s.What)
		if voice, ok := m.voices[s.Who]; ok {
			voice.SayAnything(s.What)
		} else {
			log.Printf("No voice found for %s\n", s.Who)
		}
	}
}

// Close closes the whatWasSaid channel and all the Voices.
func (l *Listener) Close() {
	close(l.whatWasSaid)
	for _, voice := range l.voices {
		voice.t.Close()
		voice.p.Close()
	}
}
