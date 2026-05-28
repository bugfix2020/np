VERSION := 1.0.0
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION)"
BUILD_DIR := dist

.PHONY: all clean build

all: build

build: linux-amd64 linux-arm64 darwin-amd64 darwin-arm64 windows-amd64 windows-arm64

linux-amd64:
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/np-linux-amd64 .

linux-arm64:
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/np-linux-arm64 .

darwin-amd64:
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/np-darwin-amd64 .

darwin-arm64:
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/np-darwin-arm64 .

windows-amd64:
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/np-windows-amd64.exe .

windows-arm64:
	GOOS=windows GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/np-windows-arm64.exe .

local:
	go build $(LDFLAGS) -o $(BUILD_DIR)/np .

clean:
	rm -rf $(BUILD_DIR)
