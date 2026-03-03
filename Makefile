.PHONY: build test lint fmt run clean doc tidy

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

doc:
	go doc $(PKG)

tidy:
	go mod tidy
