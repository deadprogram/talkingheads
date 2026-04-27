package main

import (
	"log"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type question struct {
	Content string
	To      string
}

type conversation struct {
	client    mqtt.Client
	questions chan question
}

func newConversation(server string) (*conversation, error) {
	options := mqtt.NewClientOptions()
	options.AddBroker(server)
	options.SetClientID("moderator")
	options.KeepAlive = 300

	log.Printf("Connecting to MQTT broker at %s\n", server)
	client := mqtt.NewClient(options)
	token := client.Connect()
	if token.Wait() && token.Error() != nil {
		log.Fatal("failed creating MQTT client: ", token.Error())
		return nil, token.Error()
	}

	c := &conversation{
		client:    client,
		questions: make(chan question, 1),
	}

	return c, nil
}

func (c *conversation) processQuestions() error {
	for question := range c.questions {
		topic := "ask/" + question.To

		token := c.client.Publish(topic, 0, false, []byte(question.Content))
		if token.Wait() && token.Error() != nil {
			log.Fatal("failed publishing to MQTT topic: ", token.Error())
			return token.Error()
		}
	}

	return nil
}

func (c *conversation) Close() {
	c.client.Disconnect(250)
}
