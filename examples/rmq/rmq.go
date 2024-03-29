package main

import (
	"fmt"
	"log"
	"time"

	"github.com/zalora/go-mq/rmq"
)

func main() {
	rmqURL := "amqp://rpmvioqv:6skaAUvjscD5wYNwYAHDJpokk3mniG7t@vulture.rmq.cloudamqp.com/rpmvioqv"

	log.Println("connecting to rmq host")
	connection, err := rmq.NewConnection(rmqURL,
		rmq.Config{
			ReconnectInterval: 2 * time.Second,
			Logger:            nil,
			AmqpDialler:       nil,
			ServiceName:       "",
			CommitID:          "",
			RetryForever:      true,
		})
	if err != nil {
		log.Fatal("error making new rmq connection:", err)
	}

	log.Println("connected. creating new subscriber")
	subscriber, err := rmq.NewSubscriber("go-mq-test",
		false,
		connection,
		2*time.Second)

	if err != nil {
		log.Fatal("error creating new consumer:", err)
	}

	log.Println("subscribing")
	messageCh, err := subscriber.Subscribe()
	if err != nil {
		log.Fatal("error calling consume():", err)
	}

	log.Println("listening for messages")
	for message := range messageCh {
		fmt.Println(message)
		message.Ack()
	}

}
