BINARY := fossor
MODULE := github.com/yachiko/fossor
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -X $(MODULE)/cmd.Version=$(VERSION)

.PHONY: build run test vet lint clean install deps update testdata testdata-reset tag

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

tag: ## Create and push next patch version tag (vX.Y.(Z+1))
	@set -e; \
	last=$$(git tag --list 'v*' --sort=-v:refname | head -1); \
	if [ -z "$$last" ]; then \
	  new="v0.0.1"; \
	else \
	  ver=$${last#v}; \
	  major=$${ver%%.*}; rest=$${ver#*.}; minor=$${rest%%.*}; patch=$${rest#*.}; \
	  patch=$$((patch+1)); \
	  new="v$$major.$$minor.$$patch"; \
	fi; \
	echo "Last tag: $$last"; \
	echo "New tag: $$new"; \
	git tag $$new; \
	git push origin $$new; \
	echo "✅ Created and pushed $$new"
