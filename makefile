.PHONY: run build test coverage lint clean

run:
	go run .

build:
	go build -o bin/api .

test:
	go test -v -race ./...

coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

lint:
	go vet ./...
	test -z "$$(gofmt -l .)"

clean:
	rm -rf bin coverage.out
