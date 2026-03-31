BINARY=gdrive-readonly-mcp

.PHONY: build windows darwin-amd64 darwin-arm64 linux test vet clean all

build:
	go build -o $(BINARY) .

windows:
	GOOS=windows GOARCH=amd64 go build -o $(BINARY).exe .

darwin-amd64:
	GOOS=darwin GOARCH=amd64 go build -o $(BINARY)-darwin-amd64 .

darwin-arm64:
	GOOS=darwin GOARCH=arm64 go build -o $(BINARY)-darwin-arm64 .

linux:
	GOOS=linux GOARCH=amd64 go build -o $(BINARY)-linux-amd64 .

test:
	go test ./...

vet:
	go vet ./...

clean:
	rm -f $(BINARY) $(BINARY).exe $(BINARY)-darwin-amd64 $(BINARY)-darwin-arm64 $(BINARY)-linux-amd64

all: clean build windows darwin-amd64 darwin-arm64 linux
