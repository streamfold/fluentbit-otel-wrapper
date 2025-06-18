.PHONY: build

build:
	mkdir -p dist && \
	go build -o dist/fluentbit-otel-wrapper ./cmd/fluentbit-otel-wrapper
