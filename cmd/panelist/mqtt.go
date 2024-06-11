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

type mqttInput struct {
	name      string
	questions chan llms.HumanChatMessage
	speaking  chan string
	others    chan llms.GenericChatMessage
}

func startMQTT(name, server string, questions chan llms.HumanChatMessage, speaking chan string, replies chan llms.AIChatMessage, others chan llms.GenericChatMessage) error {
	options := mqtt.NewClientOptions()
	options.AddBroker(server)
	options.SetClientID(name)
	options.KeepAlive = 300

	log.Printf("Connecting to MQTT broker at %s\n", server)
	client := mqtt.NewClient(options)
	token := client.Connect()
	if token.Wait() && token.Error() != nil {
		log.Fatal("failed creating MQTT client: ", token.Error())
		return token.Error()
	}

	m := &mqttInput{
		name:      name,
		questions: questions,
		speaking:  speaking,
		others:    others,
	}

	speak := "speak/" + name
	token = client.Subscribe(speak, 0, m.handleSpeaking)
	if token.Wait() && token.Error() != nil {
		log.Fatal("failed subscribing to MQTT topic: ", token.Error())
		return token.Error()
	}
	log.Printf("Subscribed to topic %s\n", speak)

	discuss := "discuss/" + name
	token = client.Subscribe(discuss, 0, m.handleDiscussion)
	if token.Wait() && token.Error() != nil {
		log.Fatal("failed subscribing to MQTT topic: ", token.Error())
		return token.Error()
	}
	log.Printf("Subscribed to topic %s\n", discuss)

	othersTopic := "response/#"
	token = client.Subscribe(othersTopic, 0, m.handleResponse)
	if token.Wait() && token.Error() != nil {
		log.Fatal("failed subscribing to MQTT topic: ", token.Error())
		return token.Error()
	}
	log.Printf("Subscribed to topic %s\n", othersTopic)

	reply := "response/" + name
	for {
		select {
		case r := <-replies:
			msg, err := json.Marshal(r)
			if err != nil {
				log.Println("failed marshalling message: ", err)
				return err
			}
			tok := client.Publish(reply, 0, false, msg)
			if tok.Wait() && tok.Error() != nil {
				log.Fatal("failed publishing to MQTT topic: ", tok.Error())
			}
		case <-context.Background().Done():
			return nil
		default:
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func (m *mqttInput) handleSpeaking(client mqtt.Client, msg mqtt.Message) {
	log.Printf("Received message: %s\n", string(msg.Payload()))
	m.speaking <- string(msg.Payload())
}

func (m *mqttInput) handleDiscussion(client mqtt.Client, msg mqtt.Message) {
	log.Printf("Received message: %s\n", string(msg.Payload()))
	question := llms.HumanChatMessage{}
	if err := json.Unmarshal(msg.Payload(), &question); err != nil {
		log.Println("handleDiscussion: failed unmarshalling message: ", err)
		return
	}

	m.questions <- question
}

func (m *mqttInput) handleResponse(client mqtt.Client, msg mqtt.Message) {
	log.Printf("Received on topic '%s' message: %s\n", string(msg.Topic()), string(msg.Payload()))
	from := strings.Split(msg.Topic(), "/")[1]

	if from == m.name {
		return
	}

	response := llms.AIChatMessage{}
	if err := json.Unmarshal(msg.Payload(), &response); err != nil {
		log.Println("handleResponse: failed unmarshalling message: ", err)
		return
	}

	m.others <- llms.GenericChatMessage{Content: response.Content, Name: from, Role: "Panelist"}
}
