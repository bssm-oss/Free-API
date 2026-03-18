VERSION ?= 0.1.0
BINARY = freeapi
LDFLAGS = -s -w -X github.com/bssm-oss/Free-API/cmd.Version=$(VERSION)
INSTALL_DIR = $(HOME)/.local/bin

.PHONY: build install uninstall clean test vet cross

build:
	go build -ldflags="$(LDFLAGS)" -o $(BINARY) .

install: build
	@mkdir -p $(INSTALL_DIR)
	cp $(BINARY) $(INSTALL_DIR)/$(BINARY)
	@echo "✅ Installed to $(INSTALL_DIR)/$(BINARY)"
	@echo "   Run: freeapi scan"

uninstall:
	rm -f $(INSTALL_DIR)/$(BINARY)
	@echo "Removed $(INSTALL_DIR)/$(BINARY)"

clean:
	rm -f $(BINARY) $(BINARY)-*

vet:
	go vet ./...

test:
	go test ./...

# Cross compilation
cross: cross-darwin cross-linux cross-windows

cross-darwin:
	GOOS=darwin GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o $(BINARY)-darwin-arm64 .
	GOOS=darwin GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o $(BINARY)-darwin-amd64 .

cross-linux:
	GOOS=linux GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o $(BINARY)-linux-amd64 .
	GOOS=linux GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o $(BINARY)-linux-arm64 .

cross-windows:
	GOOS=windows GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o $(BINARY)-windows-amd64.exe .
