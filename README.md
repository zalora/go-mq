# go-mq

[![Build Status](https://api.travis-ci.org/zalora/go-mq.svg?branch=master)](https://travis-ci.org/zalora/go-mq)
[![Go Report Card](https://goreportcard.com/badge/github.com/zalora/go-mq)](https://goreportcard.com/report/github.com/zalora/go-mq)
[![GoDoc](https://godoc.org/github.com/zalora/go-mq?status.svg)](https://godoc.org/github.com/zalora/go-mq)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](https://en.wikipedia.org/wiki/MIT_License)

A message queue library in Go. go-mq aims to be a high level abstraction
that provides or attempts to provide a universal consume subscribe pattern
regardless of whether the client wants to use RabbitMQ, SQS, Kafka or even
Redis.

