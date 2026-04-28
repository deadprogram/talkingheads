package actor

import (
	"encoding/json"
	"log"
	"regexp"
	"sync"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/hybridgroup/yzma/pkg/message"
)

// MQTTListener connects to an MQTT broker and wires up an Actor to receive
// user input from "ask/<name>" and publish responses to "speak/<name>".
//
// The published response payload is {"who":"<name>","what":"<content>"},
// which is compatible with the speak/# subscription in pkg/dialogue.
type MQTTListener struct {
	name      string
	client    mqtt.Client
	incoming  chan string
	done      chan struct{}
	closeOnce sync.Once
}

// NewMQTTListener connects to the broker, subscribes to "ask/<name>" for
// direct prompts and "speak/#" to hear other actors, and returns a
// ready-to-use MQTTListener.
func NewMQTTListener(name, server string) (*MQTTListener, error) {
	options := mqtt.NewClientOptions()
	options.AddBroker(server)
	options.SetClientID("actor-" + name)
	options.KeepAlive = 300

	log.Printf("Connecting to MQTT broker at %s\n", server)
	client := mqtt.NewClient(options)
	token := client.Connect()
	if token.Wait() && token.Error() != nil {
		return nil, token.Error()
	}

	l := &MQTTListener{
		name:     name,
		client:   client,
		incoming: make(chan string, 5),
		done:     make(chan struct{}),
	}

	askTopic := "ask/" + name
	token = client.Subscribe(askTopic, 0, l.handleAsk)
	if token.Wait() && token.Error() != nil {
		client.Disconnect(250)
		return nil, token.Error()
	}
	log.Printf("Subscribed to %s\n", askTopic)

	token = client.Subscribe("speak/#", 0, l.handleSpeak)
	if token.Wait() && token.Error() != nil {
		client.Disconnect(250)
		return nil, token.Error()
	}
	log.Printf("Subscribed to speak/#\n")

	return l, nil
}

func (l *MQTTListener) handleAsk(_ mqtt.Client, msg mqtt.Message) {
	text := string(msg.Payload())
	log.Printf("Received ask message: %s\n", text)
	l.enqueue(text)
}

func (l *MQTTListener) handleSpeak(_ mqtt.Client, msg mqtt.Message) {
	var s struct {
		Who  string `json:"who"`
		What string `json:"what"`
	}
	if err := json.Unmarshal(msg.Payload(), &s); err != nil {
		log.Printf("Failed to unmarshal speak message: %v\n", err)
		return
	}
	// Ignore own messages to avoid self-referential loops.
	if s.Who == l.name {
		return
	}
	log.Printf("Heard %s say: %s\n", s.Who, s.What)
	l.enqueue(s.Who + " says: " + s.What)
}

func (l *MQTTListener) enqueue(text string) {
	select {
	case l.incoming <- text:
	case <-l.done:
	}
}

// MoreFunc returns a moreConversationFunc that blocks until at least one MQTT
// message is available, then drains all buffered messages and appends each as
// a user turn. Returns without appending if the listener is closed.
func (l *MQTTListener) MoreFunc() func(*[]message.Message) {
	return func(conversation *[]message.Message) {
		// Block until the first message arrives or the listener is closed.
		select {
		case text, ok := <-l.incoming:
			if !ok || text == "" {
				return
			}
			*conversation = append(*conversation, message.Chat{Role: "user", Content: text})
		case <-l.done:
			return
		}

		// Drain any additional buffered messages without blocking.
		for {
			select {
			case text, ok := <-l.incoming:
				if !ok || text == "" {
					return
				}
				*conversation = append(*conversation, message.Chat{Role: "user", Content: text})
			default:
				return
			}
		}
	}
}

// OutputFunc returns an outputFunc that publishes the actor's response to
// "speak/<name>" as {"who":"<name>","what":"<content>"}.
func (l *MQTTListener) OutputFunc() func(string) {
	return func(content string) {
		if len(content) == 0 {
			return
		}

		content = removeEmoji(content)
		content = removeOtherUnwantedChars(content)
		payload, err := json.Marshal(struct {
			Who  string `json:"who"`
			What string `json:"what"`
		}{Who: l.name, What: content})
		if err != nil {
			log.Printf("failed to marshal response: %v\n", err)
			return
		}

		topic := "speak/" + l.name
		token := l.client.Publish(topic, 0, false, payload)
		token.Wait()
		if token.Error() != nil {
			log.Printf("failed to publish response to %s: %v\n", topic, token.Error())
		}
	}
}

// Close unblocks any pending MoreFunc call and disconnects from the broker.
// Safe to call multiple times.
func (l *MQTTListener) Close() {
	l.closeOnce.Do(func() {
		close(l.done)
		l.client.Disconnect(250)
	})
}

func removeEmoji(str string) string {
	// Regex pattern to match most emoji characters
	emojiPattern := "[\U0001F600-\U0001F64F\U0001F300-\U0001F5FF\U0001F680-\U0001F6FF\U0001F700-\U0001F77F\U0001F780-\U0001F7FF\U0001F800-\U0001F8FF\U0001F900-\U0001F9FF\U00002702-\U000027B0\U000024C2-\U0001F251]+"
	re := regexp.MustCompile(emojiPattern)
	// Replace matched emoji with an empty string to remove it
	return re.ReplaceAllString(str, "")
}

func removeOtherUnwantedChars(str string) string {
	// Regex pattern to match control characters and symbols that cannot be read aloud
	// (e.g., markdown formatting: *, _, #, `, ~, |, \).
	unwantedPattern := `[\x00-\x1F\x7F*_#` + "`~|\\\\]+"
	re := regexp.MustCompile(unwantedPattern)
	return re.ReplaceAllString(str, "")
}
