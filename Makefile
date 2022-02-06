GO          := go
GOFMT       := gofmt
GOLINT      := golint
GO111MODULE := on
GOHOSTOS    := $(shell $(GO) env GOHOSTOS)
GOHOSTARCH  := $(shell $(GO) env GOHOSTARCH)

MODULE := $(shell env GO111MODULE=on $(GO) list -m)

BUILD_DATE     := $(shell date '+%FT%T')
BUILD_REVISION := $(shell git rev-parse HEAD)
BUILD_VERSION  := $(shell git describe --tags --always 2> /dev/null)

LDFLAGS=-ldflags "-X $(MODULE)/build.Date=$(BUILD_DATE) -X $(MODULE)/build.Revision=$(BUILD_REVISION) -X $(MODULE)/build.Version=$(BUILD_VERSION)"

.PHONY: check
check:
	$(GOFMT) -s -d .
	$(GOLINT) ./...
	$(GO) vet ./...

.PHONY: build
build:
	GO111MODULE=$(GO111MODULE) GOOS=$(GOHOSTOS) GOARCH=$(GOHOSTARCH) $(GO) get -d ./...
	GO111MODULE=$(GO111MODULE) GOOS=$(GOHOSTOS) GOARCH=$(GOHOSTARCH) $(GO) build $(LDFLAGS) -o ./github_exporter ./
