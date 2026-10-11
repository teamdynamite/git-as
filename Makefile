BIN := git-as
INSTALL_DIR := $(or $(GOBIN),$(HOME)/bin)

.PHONY: build test install clean

build:
	go build -o $(BIN) .

test:
	go test ./...

install: build
	mkdir -p $(INSTALL_DIR)
	install -m 755 $(BIN) $(INSTALL_DIR)/$(BIN)

clean:
	rm -f $(BIN)
