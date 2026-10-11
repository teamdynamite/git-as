BIN := git-as
INSTALL_DIR ?= $(HOME)/.local/bin

.PHONY: build test install clean

# build keeps the symbol table and debug info for development.
build:
	go build -o $(BIN) .

test:
	go test ./...

# install strips the symbol table and debug info (-s -w) to make the binary smaller.
install:
	go build -ldflags="-s -w" -o $(BIN) .
	mkdir -p $(INSTALL_DIR)
	install -m 755 $(BIN) $(INSTALL_DIR)/$(BIN)

clean:
	rm -f $(BIN)
