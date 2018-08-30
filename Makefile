SHELL = /bin/bash

TARGETS = clinker

all: $(TARGETS)

$(TARGETS): %: cmd/%/main.go
	go get -v ./...
	go build -ldflags="-s -w" -v -o $@ $<

clean:
	rm -f $(TARGETS)
