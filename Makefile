BINARY  := sprint-ledger
PREFIX  ?= $(HOME)/.local

.PHONY: build test install uninstall clean

build:
	go build -o $(BINARY) .

test:
	go test ./... -v

install: build
	install -d $(PREFIX)/bin
	install -m 755 $(BINARY) $(PREFIX)/bin/$(BINARY)

uninstall:
	rm -f $(PREFIX)/bin/$(BINARY)

clean:
	rm -f $(BINARY)
