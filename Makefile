VERSION ?= $(shell git describe --tags --always 2>/dev/null)

build:
	go build -ldflags "-X main.Version=$(VERSION)" -o ./bin/nezabx

clean:
	rm -rf ./bin