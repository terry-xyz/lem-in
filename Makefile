BINDIR := bin
WEB_FILE := $(or $(FILE),$(firstword $(filter %.txt,$(MAKECMDGOALS))))
OUT ?= colony.html

.PHONY: build run viz-tui viz-web viz-web-open test bench fuzz lint fmt coverage clean $(WEB_FILE)

build:
	go build -o $(BINDIR)/lem-in ./cmd/lem-in
	go build -o $(BINDIR)/visualizer-tui ./cmd/visualizer-tui
	go build -o $(BINDIR)/visualizer-web ./cmd/visualizer-web

run:
	go run ./cmd/lem-in $(FILE)

viz-tui:
	go run ./cmd/lem-in $(FILE) | go run ./cmd/visualizer-tui

viz-web:
	@if [ -z "$(WEB_FILE)" ]; then echo "usage: make viz-web FILE=examples/example01.txt"; echo "   or: make viz-web examples/example01.txt"; exit 2; fi
	@go run ./cmd/lem-in $(WEB_FILE) | go run ./cmd/visualizer-web > $(OUT)
	@echo "Generated $(OUT)" >&2

viz-web-open:
	@if [ -z "$(WEB_FILE)" ]; then echo "usage: make viz-web-open FILE=examples/example01.txt"; echo "   or: make viz-web-open examples/example01.txt"; exit 2; fi
	@go run ./cmd/lem-in $(WEB_FILE) | go run ./cmd/visualizer-web > $(OUT)
	@if command -v xdg-open >/dev/null 2>&1 && xdg-open "$(OUT)" >/dev/null 2>&1; then :; \
	elif command -v open >/dev/null 2>&1 && open "$(OUT)" >/dev/null 2>&1; then :; \
	elif command -v wslview >/dev/null 2>&1 && wslview "$(OUT)" >/dev/null 2>&1; then :; \
	elif command -v explorer.exe >/dev/null 2>&1 && explorer.exe "$(OUT)" >/dev/null 2>&1; then :; \
	elif command -v cmd.exe >/dev/null 2>&1 && cmd.exe /C start "" "$(OUT)" >/dev/null 2>&1; then :; \
	else echo "Generated $(OUT). Open it in a browser."; fi

test:
	go test -p 1 ./...

bench:
	go test -bench=. ./...

fuzz:
	go test -fuzz=Fuzz -fuzztime=30s ./internal/parser/

lint:
	go vet ./...

fmt:
	gofmt -l .

coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

clean:
	rm -rf $(BINDIR) coverage.out coverage.html

ifneq ($(WEB_FILE),)
$(WEB_FILE):
	@true
endif
