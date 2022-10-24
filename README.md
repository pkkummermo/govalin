# govalin

[![Unit tests](https://github.com/pkkummermo/govalin/actions/workflows/main.yml/badge.svg)](https://github.com/pkkummermo/govalin/actions/workflows/main.yml)

A simple way of creating efficient HTTP APIs in golang using conventions over configuration.

## Installation

To install govalin run:

```bash
go get -u github.com/pkkummermo/govalin
```

## Hello World

```go
func main() {
	govalin.New().
		Get("/test", func(call *govalin.Call) {
			call.Text("Hello world")
		}).
		Start(7070)
}
```

## Motivation

I love how fast and efficient go is. What I don't like, is how it doesn't create an easy way of creating HTTP APIs. Govalin focuses on pleasing those who want to create APIs without too much hassle, with a lean simple API.

Inspired by simple libraries and frameworks such as [Javalin](https://javalin.io), I wanted to see if we could port the simplicity to golang.
