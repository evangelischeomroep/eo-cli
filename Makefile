.PHONY: tidy test vet build check

BUILD_OUTPUT ?= /tmp/eo

tidy:
	go mod tidy

test:
	go test ./...

vet:
	go vet ./...

build:
	go build -o $(BUILD_OUTPUT) ./cmd/eo

check: tidy
	$(MAKE) test
	$(MAKE) vet
	$(MAKE) build
