VERSION := $(shell git describe --tags)
BUILD := $(shell git rev-parse --short HEAD)
LDFLAGS := -ldflags "-X=main.Version=$(VERSION) -X=main.Build=$(BUILD)"
ARCH_LIST = darwin linux windows

.PHONY: test release

test:
	go test ./pkg/...

release: $(patsubst %, release/aggspread-%-amd64.tar.gz, $(ARCH_LIST))

release/aggspread-%-amd64.tar.gz: release/aggspread-%-amd64
	tar -czvf $@ $<

.PRECIOUS: release/aggspread-%-amd64
release/aggspread-%-amd64:
	mkdir -p $@
	cp README.md $@
	cp LICENSE $@
	GOOS=$* GOARCH=amd64 go build $(LDFLAGS) -o $@/aggspread main.go
