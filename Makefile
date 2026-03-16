.PHONY: build test run dev clean

build:
	cd core && go build -o ../bin/magic ./cmd/magic

test:
	cd core && go test ./... -v

run: build
	./bin/magic serve

dev:
	cd core && go run ./cmd/magic serve

clean:
	rm -rf bin/
