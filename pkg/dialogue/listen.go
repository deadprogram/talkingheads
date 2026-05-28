package dialogue

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/talkingheads2053/talkingheads/pkg/commands"
)

// Listener listens for MQTT messages with something that was said, and then tells the appropriate Voice to say it.
type Listener struct {
	client      mqtt.Client
	whatWasSaid chan commands.Speak
	voices      map[string]*Voice
	verbose     bool
	publishCh   chan publishMsg
	eventsCh    chan<- string
}

// SetEventsCh registers a channel that receives human-readable event strings
// (e.g. speech dispatches, warnings). Must be called before Listen().
func (m *Listener) SetEventsCh(ch chan<- string) {
	m.eventsCh = ch
}

// emit sends a message to eventsCh when it is set, without blocking.
func (m *Listener) emit(msg string) {
	if m.eventsCh == nil {
		return
	}
	select {
	case m.eventsCh <- msg:
	default:
	}
}

type publishMsg struct {
	topic   string
	payload []byte
}

// NewListener starts the MQTT client and subscribes to the topic for something that was said.
// It returns a Listener that can be used to listen for messages and tell the appropriate Voice to say it.
func NewListener(name, server string, voices map[string]*Voice, verbose bool) (*Listener, error) {
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
		verbose:     verbose,
		publishCh:   make(chan publishMsg, 32),
	}

	// Single publisher goroutine: drains publishCh in FIFO order so that
	// StatusStopped is always delivered before the next StatusSpeaking,
	// regardless of how quickly back-to-back phrases are queued.
	go func() {
		for msg := range m.publishCh {
			token := m.client.Publish(msg.topic, 0, false, msg.payload)
			token.Wait()
			if token.Error() != nil {
				log.Printf("Failed to publish speaking message to %s: %v\n", msg.topic, token.Error())
			}
		}
	}()

	speakTopic := "speak/#"
	token = client.Subscribe(speakTopic, 0, m.handleSpeaking)
	if token.Wait() && token.Error() != nil {
		log.Fatal("failed subscribing to MQTT topic: ", token.Error())
		return nil, token.Error()
	}
	log.Printf("Subscribed to topic %s\n", speakTopic)

	sayTopic := "say/#"
	token = client.Subscribe(sayTopic, 0, m.handleSay)
	if token.Wait() && token.Error() != nil {
		log.Fatal("failed subscribing to MQTT topic: ", token.Error())
		return nil, token.Error()
	}
	log.Printf("Subscribed to topic %s\n", sayTopic)

	return m, nil
}

func (m *Listener) handleSpeaking(client mqtt.Client, msg mqtt.Message) {
	if m.verbose {
		log.Printf("Received message: %s\n", string(msg.Payload()))
	}

	// parse the payload and extract the text to speak
	var s commands.Speak
	if err := json.Unmarshal(msg.Payload(), &s); err != nil {
		log.Printf("Failed to unmarshal message: %v\n", err)
		return
	}
	m.whatWasSaid <- s
}

// handleSay handles messages on the say/# topic. The payload is identical in
// shape to a Speak message, but Actors do not subscribe to say/#, so the
// utterance is spoken (and a speaking notification is published) without being
// added to any Actor's conversation history.
func (m *Listener) handleSay(client mqtt.Client, msg mqtt.Message) {
	if m.verbose {
		log.Printf("Received say message: %s\n", string(msg.Payload()))
	}

	var s commands.Say
	if err := json.Unmarshal(msg.Payload(), &s); err != nil {
		log.Printf("Failed to unmarshal say message: %v\n", err)
		return
	}
	m.whatWasSaid <- commands.Speak{Who: s.Who, What: s.What}
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

			done := make(chan struct{})
			go func(who string) {
				ticker := time.NewTicker(200 * time.Millisecond)
				defer ticker.Stop()
				for {
					select {
					case <-ticker.C:
						m.publishSpeaking(who, commands.StatusSpeaking)
					case <-done:
						return
					}
				}
			}(req.who)

			req.voice.SayOnce(req.what)
			close(done)
			m.publishSpeaking(req.who, commands.StatusStopped)
		}
	}()

	for s := range m.whatWasSaid {
		if m.verbose {
			log.Printf("Received something said: %s\n", s.What)
		}
		if voice, ok := m.voices[s.Who]; ok {
			m.emit(fmt.Sprintf("→ %s: %q", s.Who, s.What))
			queue <- playRequest{voice: voice, who: s.Who, what: s.What}
		} else {
			log.Printf("WARNING: No voice found for %s\n", s.Who)
			m.emit(fmt.Sprintf("WARNING: no voice for %q", s.Who))
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
	msg := publishMsg{topic: "speaking/" + who, payload: payload}
	if m.publishCh != nil {
		m.publishCh <- msg
		return
	}
	// publishCh is nil (e.g. bare struct literal in tests): publish synchronously.
	token := m.client.Publish(msg.topic, 0, false, msg.payload)
	token.Wait()
	if token.Error() != nil {
		log.Printf("Failed to publish speaking message to %s: %v\n", msg.topic, token.Error())
	}
}

// Close closes the whatWasSaid channel and all the Voices.
func (l *Listener) Close() {
	close(l.whatWasSaid)
	close(l.publishCh)
	for _, voice := range l.voices {
		voice.t.Close()
		voice.p.Close()
	}
}
