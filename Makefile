.PHONY: all clean install

all: quotes-plugin dulbecco

quotes-plugin: quotes/*.go cmd/quotes-plugin/quotes-plugin.go
	go build ./cmd/quotes-plugin

dulbecco: *.go cmd/dulbecco/dulbecco.go markov/*.go
	go build ./cmd/dulbecco

clean:
	-rm dulbecco
	-rm quotes-plugin

install:
	go install ./...
