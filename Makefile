.PHONY: build test lint fmt run clean

BINARY := cicada

build:
	go build -o $(BINARY) .

test:
	go test ./... -v

lint:
	go vet ./...

fmt:
	gofmt -w .

run: build
	./$(BINARY)

clean:
	rm -f $(BINARY)
