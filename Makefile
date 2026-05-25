BINARY := fossor
MODULE := github.com/ahoma/fossor
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -X $(MODULE)/cmd.Version=$(VERSION)

.PHONY: build run test vet lint clean install deps update testdata testdata-reset

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) .

run: build
	./$(BINARY)

install:
	go install -ldflags "$(LDFLAGS)" .

test:
	go test ./...

vet:
	go vet ./...

lint: vet
	@echo "All checks passed"

deps:
	go mod tidy

update:
	go get -u ./...
	go mod tidy

clean:
	rm -f $(BINARY)
	rm -rf testdata/repos testdata/remotes

testdata:
	bash testdata/setup.sh

testdata-reset:
	bash testdata/reset.sh

check: vet test build
	@echo "Build OK"
