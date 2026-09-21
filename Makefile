.PHONY: bootstrap build check doctor fmt test

BUILD_DIR ?= bin

bootstrap:
	brew bundle install --no-upgrade

fmt:
	gofmt -w cmd internal

test:
	go test ./...

build:
	mkdir -p "$(BUILD_DIR)"
	go build -o "$(BUILD_DIR)/code-remote-daemon" ./cmd/daemon

check:
	@unformatted="$$(gofmt -l cmd internal)"; \
	if [ -n "$$unformatted" ]; then \
		echo "Go files need formatting:"; \
		echo "$$unformatted"; \
		exit 1; \
	fi
	go test ./...
	go vet ./...
	sh scripts/development_setup_test.sh

doctor:
	./scripts/doctor.sh
