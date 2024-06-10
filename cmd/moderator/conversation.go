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
	questions chan question //llms.HumanChatMessage
	responses chan llms.AIChatMessage
}

func startConversation(server string) (*conversation, error) {
	options := mqtt.NewClientOptions()
	options.AddBroker(server)
	options.SetClientID("moderator")

	log.Printf("Connecting to MQTT broker at %s\n", server)
	client := mqtt.NewClient(options)
	token := client.Connect()
	if token.Wait() && token.Error() != nil {
		log.Fatal("failed creating MQTT client: ", token.Error())
		return nil, token.Error()
	}

	c := &conversation{
		client:    client,
		questions: make(chan question),
		responses: make(chan llms.AIChatMessage),
	}

	responseTopic := "response/#"
	token = client.Subscribe(responseTopic, 0, c.handleResponse)
	if token.Wait() && token.Error() != nil {
		log.Fatal("failed subscribing to MQTT topic: ", token.Error())
		return nil, token.Error()
	}
	log.Printf("Subscribed to topic %s\n", responseTopic)

	go c.handleQuestions()

	return c, nil
}

func (c *conversation) handleResponse(client mqtt.Client, msg mqtt.Message) {
	log.Printf("Received message: %s\n", string(msg.Payload()))
	response := llms.AIChatMessage{}
	if err := json.Unmarshal(msg.Payload(), &response); err != nil {
		log.Println("failed unmarshalling message: ", err)
		return
	}

	c.responses <- response
}

func (c *conversation) handleQuestions() error {
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
