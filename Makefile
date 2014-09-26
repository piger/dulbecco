GO_FILES := $(shell find . -depth 1 -name '*.go')
QUOTES_FILES := $(shell find quotes -name '*.go')

all: quotes-plugin dulbecco

quotes-plugin: $(QUOTA_FILES) cmd/quotes-plugin/quotes-plugin.go
	go build -tags "libstemmer icu" cmd/quotes-plugin/quotes-plugin.go

dulbecco: $(GO_FILES) cmd/dulbecco/dulbecco.go markov/markov.go
	go build cmd/dulbecco/dulbecco.go


.PHONY: all
