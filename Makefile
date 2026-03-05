BINDIR := bin

.PHONY: build run viz-tui viz-web test bench fuzz lint fmt coverage clean

build:
	go build -o $(BINDIR)/lem-in ./cmd/lem-in
	go build -o $(BINDIR)/visualizer-tui ./cmd/visualizer-tui
	go build -o $(BINDIR)/visualizer-web ./cmd/visualizer-web

run:
	go run ./cmd/lem-in $(FILE)

viz-tui:
	go run ./cmd/lem-in $(FILE) | go run ./cmd/visualizer-tui

viz-web:
	go run ./cmd/lem-in $(FILE) | go run ./cmd/visualizer-web

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
