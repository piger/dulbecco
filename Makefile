.PHONY: all clean install

all: quotes-plugin dulbecco convert-db

quotes-plugin: quotes/*.go cmd/quotes-plugin/quotes-plugin.go
	go build ./cmd/quotes-plugin

dulbecco: *.go cmd/dulbecco/dulbecco.go markov/*.go
	go build ./cmd/dulbecco

convert-db: cmd/convert-db/convert-db.go
	go build ./cmd/convert-db

clean:
	-rm dulbecco
	-rm quotes-plugin
	-rm convert-db

install:
	go install ./...
