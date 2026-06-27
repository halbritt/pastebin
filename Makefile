.PHONY: build test clean

build:
	go build -o bin/pastebind ./cmd/pastebind
	go build -o bin/pastebin ./cmd/pastebin

test:
	go test ./...

clean:
	rm -rf bin coverage.out
