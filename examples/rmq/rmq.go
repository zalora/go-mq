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
		})
	if err != nil {
		log.Fatal(err, "error making new rmq connection")
	}

	log.Println("opening new channel")
	ch, err := connection.NewChannel()
	if err != nil {
		log.Fatal(err, "error connecting to new channel")
	}

	fmt.Println(ch)
	time.Sleep(10 * time.Second)
}
