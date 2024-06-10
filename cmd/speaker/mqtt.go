package main

import (
	"encoding/json"
	"log"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/tmc/langchaingo/llms"
)

type mqttInput struct {
	questions chan llms.HumanChatMessage
	speaking  chan string
}

// TODO: implement this function
func startMQTT(name, server string, questions chan llms.HumanChatMessage, speaking chan string, replies chan llms.AIChatMessage) error {
	options := mqtt.NewClientOptions()
	options.AddBroker(server)
	options.SetClientID(name)

	log.Printf("Connecting to MQTT broker at %s\n", server)
	client := mqtt.NewClient(options)
	token := client.Connect()
	if token.Wait() && token.Error() != nil {
		log.Fatal("failed creating MQTT client: ", token.Error())
		return token.Error()
	}

	m := &mqttInput{
		questions: questions,
		speaking:  speaking,
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

	reply := "response/" + name
	for r := range replies {
		msg, err := json.Marshal(r)
		if err != nil {
			log.Println("failed marshalling message: ", err)
			return err
		}
		token = client.Publish(reply, 0, false, msg)
		if token.Wait() && token.Error() != nil {
			log.Fatal("failed publishing to MQTT topic: ", token.Error())
			return token.Error()
		}
	}

	return nil
}

func (m *mqttInput) handleSpeaking(client mqtt.Client, msg mqtt.Message) {
	log.Printf("Received message: %s\n", string(msg.Payload()))
	m.speaking <- string(msg.Payload())
}

func (m *mqttInput) handleDiscussion(client mqtt.Client, msg mqtt.Message) {
	log.Printf("Received message: %s\n", string(msg.Payload()))
	question := llms.HumanChatMessage{}
	if err := json.Unmarshal(msg.Payload(), &question); err != nil {
		log.Println("failed unmarshalling message: ", err)
		return
	}

	m.questions <- question
}
