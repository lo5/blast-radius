.PHONY: build install test clean

build:
	go build -o bin/blast-radius ./cmd/blast-radius

install:
	go install ./cmd/blast-radius

test:
	go test ./...

clean:
	rm -rf bin
