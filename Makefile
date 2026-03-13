BINARY   := n8n-cli
PREFIX   := /usr/local/bin
GOFLAGS  := -trimpath
LDFLAGS  := -s -w

.PHONY: build install uninstall clean test vet fmt check

build:
	go build $(GOFLAGS) -ldflags '$(LDFLAGS)' -o $(BINARY) .

install: build
	sudo install -m 755 $(BINARY) $(PREFIX)/$(BINARY)

uninstall:
	sudo rm -f $(PREFIX)/$(BINARY)

clean:
	rm -f $(BINARY)

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -w .

check: vet test
	@echo "All checks passed."
