package main

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/tmc/langchaingo/llms"
)

type question struct {
	Content string `json:"content"`
	To      string `json:"to"`
}

type response struct {
	Content string `json:"content"`
	From    string `json:"from"`
}

type conversation struct {
	client    mqtt.Client
	questions chan question
	responses chan response
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
		questions: make(chan question),
		responses: make(chan response),
	}

	return c, nil
}

func (c *conversation) processResponses() error {
	responseTopic := "response/#"
	token := c.client.Subscribe(responseTopic, 0, c.handleResponse)
	if token.Wait() && token.Error() != nil {
		log.Fatal("failed subscribing to MQTT topic: ", token.Error())
		return token.Error()
	}
	log.Printf("Subscribed to topic %s\n", responseTopic)

	for {
		select {
		case <-context.Background().Done():
			return nil
		default:
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func (c *conversation) handleResponse(client mqtt.Client, msg mqtt.Message) {
	log.Printf("Received message: %s\n", string(msg.Payload()))
	airesponse := llms.AIChatMessage{}
	if err := json.Unmarshal(msg.Payload(), &airesponse); err != nil {
		log.Println("failed unmarshalling message: ", err)
		return
	}

	from := strings.Split(msg.Topic(), "/")[1]
	c.responses <- response{From: from, Content: airesponse.Content}
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
