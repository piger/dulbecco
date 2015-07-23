.PHONY: all

all: quotes-plugin dulbecco

quotes-plugin: quotes/*.go cmd/quotes-plugin/quotes-plugin.go
	go build -tags "libstemmer" cmd/quotes-plugin/quotes-plugin.go

dulbecco: *.go cmd/dulbecco/dulbecco.go markov/markov.go
	go build cmd/dulbecco/dulbecco.go
