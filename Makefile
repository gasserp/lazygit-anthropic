.PHONY: build install test vet fmt clean

build:
	go build -o bin/lazygit-ai .

install:
	go install .

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -w .

clean:
	rm -rf bin/
