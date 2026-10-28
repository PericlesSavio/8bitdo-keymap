MODULE = github.com/PericlesSavio/8bitdoctl
VERSION = 0.5.0

GO := go
GO_FLAGS = -ldflags "-X '$(MODULE)/internal/version.Version=$(VERSION)'"

build:
	CGO_ENABLED=0 $(GO) build $(GO_FLAGS) ./cmd/8bitdod
	CGO_ENABLED=0 $(GO) build $(GO_FLAGS) ./cmd/8bitdoctl

test:
	$(GO) vet ./...
	$(GO) test ./...

install:
	./install.sh

uninstall:
	./install.sh --uninstall

clean:
	rm -f 8bitdod 8bitdoctl 8bitdoctl.bash

.PHONY: build test install uninstall clean
