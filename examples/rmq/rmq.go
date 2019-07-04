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
		log.Fatal(err, "error making new rmq connection")
	}

	consumer, err := rmq.NewConsumer("go-mq-test",
		false,
		connection,
		2*time.Second)

	if err != nil {
		log.Fatal(err, "error creating new consumer")
	}

	messageCh, err := consumer.Consume()
	if err != nil {
		log.Fatal(err, "error calling consume()")
	}

	for message := range messageCh {
		fmt.Println(message)
		message.Ack()
	}

}
