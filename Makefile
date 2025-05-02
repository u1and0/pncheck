### Usage
#
# ```bash
# make     		     # Build all
# make all 		     # Build all
# make build         # Build with default server address
# make exe           # Build for Windows
# make doc           # Create documentation
# make clean         # Clean up
# ```
#
# You can override the `SERVER_ADDRESS` variable when running the `build` target:
#
# ```bash
# make build SERVER_ADDRESS=http://example.com:8080
# ```

BUILD_CMD = go build -ldflags="-X pncheck/lib/input.serverAddress=$(SERVER_ADDRESS)"
SERVER_ADDRESS ?= http://localhost:8080

all: build exe doc

.PHONY: all build exe doc clean

build:
	$(BUILD_CMD)
exe:
	GOOS=windows GOARCH=amd64 $(BUILD_CMD)
doc:
	pandoc README.md -o README.html
clean:
	rm -f pncheck pncheck.exe README.html
