.PHONY: build build-local test vet fmt clean

BINARY := fleet-pulse

# The deploy target is a bare-metal Linux host running systemd, so `make
# build` always cross-compiles for linux/amd64 regardless of the machine
# it's run on -- including this macOS dev machine.
build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $(BINARY) .

# For running the agent directly on the current machine during development
# (e.g. `go run .` equivalent, but as a built binary).
build-local:
	CGO_ENABLED=0 go build -o $(BINARY) .

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -l .

clean:
	rm -f $(BINARY)
