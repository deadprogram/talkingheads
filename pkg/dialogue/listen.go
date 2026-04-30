package dialogue

import (
	"encoding/json"
	"errors"
	"log"

	"github.com/deadprogram/talkingheads/pkg/commands"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// Listener listens for MQTT messages with something that was said, and then tells the appropriate Voice to say it.
type Listener struct {
	client      mqtt.Client
	whatWasSaid chan commands.Speak
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
		client:      client,
		whatWasSaid: make(chan commands.Speak, 5),
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
	var s commands.Speak
	if err := json.Unmarshal(msg.Payload(), &s); err != nil {
		log.Printf("Failed to unmarshal message: %v\n", err)
		return
	}
	m.whatWasSaid <- s
}

// Listen listens for messages on the whatWasSaid channel and tells the appropriate Voice to say it.
// Audio playback is serialized through a single worker goroutine so that the underlying
// audio library context is never called concurrently.
func (m *Listener) Listen() {
	type playRequest struct {
		voice *Voice
		who   string
		what  string
	}
	queue := make(chan playRequest, 10)

	// Single worker: plays speech requests one at a time.
	go func() {
		for req := range queue {
			m.publishSpeaking(req.who, commands.StatusSpeaking)
			req.voice.SayOnce(req.what)
			m.publishSpeaking(req.who, commands.StatusStopped)
		}
	}()

	for s := range m.whatWasSaid {
		log.Printf("Received something said: %s\n", s.What)
		if voice, ok := m.voices[s.Who]; ok {
			queue <- playRequest{voice: voice, who: s.Who, what: s.What}
		} else {
			log.Printf("No voice found for %s\n", s.Who)
		}
	}
	close(queue)
}

func (m *Listener) publishSpeaking(who, status string) {
	payload, err := json.Marshal(commands.Speaking{Who: who, Status: status})
	if err != nil {
		log.Printf("Failed to marshal speaking message: %v\n", err)
		return
	}
	topic := "speaking/" + who
	token := m.client.Publish(topic, 0, false, payload)
	token.Wait()
	if token.Error() != nil {
		log.Printf("Failed to publish speaking message to %s: %v\n", topic, token.Error())
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
