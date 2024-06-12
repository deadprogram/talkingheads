package main

import (
	"encoding/json"
	"log"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/tmc/langchaingo/llms"
)

type question struct {
	Content string `json:"content"`
	To      string `json:"to"`
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
		name := question.To
		discuss := "discuss/" + name

		msg, err := json.Marshal(llms.HumanChatMessage{Content: question.Content})
		if err != nil {
			log.Println("failed marshalling message: ", err)
			return err
		}
		token := c.client.Publish(discuss, 0, false, msg)
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
