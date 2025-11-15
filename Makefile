all: format build test

format:
	gofumpt -w $$(find . -name '*.go')

build:
	go build -v ./...

test:
	go test -v ./...

install:
	sudo install gcl /usr/local/bin

update:
	go get -u
	go mod tidy
