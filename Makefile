GO_FILES := $(shell find . -depth 1 -name '*.go')
QUOTES_FILES := $(shell find quotes -name '*.go')

all: quotes-plugin dulbecco

quotes-plugin: $(QUOTES_FILES)
	go build -o quotes-plugin -tags "libstemmer icu" ./quotes

dulbecco: $(GO_FILES)
	go build cmd/dulbecco/dulbecco.go


.PHONY: all
