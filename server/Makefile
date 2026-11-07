.PHONY: build test vet run clean

build:
	go build -o bin/server ./cmd/server

test:
	go test ./... -count=1

vet:
	go vet ./...

run:
	DB_PATH=:memory: HTTP_ADDR=:8080 go run ./cmd/server

clean:
	rm -rf bin/ server_test.exe

lint: vet test
