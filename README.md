# go-mq

[![CircleCI](https://dl.circleci.com/status-badge/img/gh/zalora/go-mq/tree/master.svg?style=shield)](https://dl.circleci.com/status-badge/redirect/gh/zalora/go-mq/tree/master)
[![Go Report Card](https://goreportcard.com/badge/github.com/zalora/go-mq)](https://goreportcard.com/report/github.com/zalora/go-mq)
[![GoDoc](https://godoc.org/github.com/zalora/go-mq?status.svg)](https://godoc.org/github.com/zalora/go-mq)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](https://en.wikipedia.org/wiki/MIT_License)

A message queue library in Go. go-mq aims to be a high level abstraction
that provides or attempts to provide a universal consume subscribe pattern
regardless of whether the client wants to use RabbitMQ, SQS, Kafka or even
Redis.

## Requirements

 - Go 1.19+

## Usage

Pull in the library or functionality you want to use via

```
go get github.com/zalora/go-mq/...
```

Check the [examples](./examples).

## Contributing

If you find any issues or missing a feature, feel free to contribute or make 
suggestions. You can fork the repository and use a feature branch too. Feel free
to send a pull request. The PRs have to come with appropriate unit tests,
documentation of the added functionality and updated README with optional
examples.

To start developing clone via `git` or use go's get command to fetch this 
project.

This project uses [go modules](https://github.com/golang/go/wiki/Modules) so
make sure when adding new dependencies to update the `go.mod` and keep it clean:

```
go mod tidy
```

## Licensing

The code in this project is licensed under MIT license.
