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
	connection, err := rmq.NewConnection(rmqURL, 5*time.Second, nil, nil, "exampleService", "1123")
	if err != nil {
		log.Fatal(err, "error making new rmq connection")
	}

	log.Println("opening new channel")
	ch, err := connection.NewChannel()
	if err != nil {
		log.Fatal(err, "error connecting to new channel")
	}

	fmt.Println(ch)
}
