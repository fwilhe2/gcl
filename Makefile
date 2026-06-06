VERSION ?= dev
LDFLAGS := -X github.com/fwilhe2/gcl/cmd.version=$(VERSION)

all: format build test

format:
	gofumpt -w $$(find . -name '*.go')

build:
	go build -v -ldflags "$(LDFLAGS)" -o gcl main.go

test:
	go test -v ./...

install:
	sudo install gcl /usr/local/bin

update:
	go get -u
	go mod tidy
