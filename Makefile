APP := multicard-mcp-go
BIN_DIR := bin
DIST_DIR := dist
PACKAGE_DIR := $(DIST_DIR)/package

.PHONY: fmt test build build-http clean package-linux-amd64 run-http run-stdio

fmt:
	gofmt -w *.go

test:
	go test ./...

build: fmt test
	mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/$(APP) .

build-http: build

run-http:
	go run . --http --listen-addr 127.0.0.1:8080

run-stdio:
	go run .

package-linux-amd64: fmt test
	rm -rf $(PACKAGE_DIR)
	mkdir -p $(PACKAGE_DIR)/bin
	GOOS=linux GOARCH=amd64 go build -ldflags='-s -w' -o $(PACKAGE_DIR)/bin/$(APP) .
	cp -a multicard-docs $(PACKAGE_DIR)/
	cp -a deploy $(PACKAGE_DIR)/
	cp -a scripts $(PACKAGE_DIR)/
	cp README.md $(PACKAGE_DIR)/
	mkdir -p $(DIST_DIR)
	tar -C $(PACKAGE_DIR) -czf $(DIST_DIR)/$(APP)-linux-amd64.tar.gz .

clean:
	rm -rf $(BIN_DIR) $(DIST_DIR)
