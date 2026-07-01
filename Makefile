.PHONY: build install test vet fmt clean

build:
	go build -o bin/lazygit-ai ./cmd/lazygit-ai

install:
	go install ./cmd/lazygit-ai

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -w .

clean:
	rm -rf bin/
