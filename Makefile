BIN := gitpad
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

.PHONY: build run test lint install snapshot demo-repo media release-check clean

DEMO_DIR ?= /tmp/gitpad-demo

build:
	go build -ldflags "-s -w -X main.version=$(VERSION)" -o $(BIN) .

run: build
	./$(BIN)

test:
	go test ./...

lint:
	go vet ./...

install:
	go install -ldflags "-s -w -X main.version=$(VERSION)" .

# Render one frame of the current repo as text (layout debugging without a TTY).
snapshot: build
	./$(BIN) --snapshot=160x45 . | sed 's/\x1b\[[0-9;]*m//g'

# Synthetic, anonymous repository used for screenshots (~400 commits).
demo-repo:
	go run ./scripts/demo-repo $(DEMO_DIR)

# Record docs/media/*.gif and docs/screenshots/*.png with vhs (brew install vhs).
media:
	DEMO_DIR=$(DEMO_DIR) scripts/record-media.sh


release-check:
	goreleaser check
	goreleaser build --snapshot --clean --single-target

clean:
	rm -f $(BIN)
	rm -rf dist
