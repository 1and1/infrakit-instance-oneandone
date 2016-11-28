# REPO
REPO?=github.com/StackPointCloud/infrakit-instance-oneandone

# Used to populate version variable in main package.
VERSION?=$(shell cat -A version)
REVISION?=$(shell git rev-list -1 HEAD)

# Allow turning off function inlining and variable registerization
ifeq (${DISABLE_OPTIMIZATION},true)
	GO_GCFLAGS=-gcflags "-N -l"
	VERSION:="$(VERSION)-noopt"
endif

.PHONY: clean default build test
.DEFAULT: all
default: build test

# Package list
PKGS := $(shell echo $(shell go list ./... | grep -v ^${REPO}/vendor/) | tr ' ' '\n')

build:
	@echo "+ $@"
	@go build ${GO_LDFLAGS} $(PKGS)

clean:
	@echo "+ $@"
	rm -rf build
	mkdir -p build

binary: clean
	@echo "+ $@"
	@go build -o ./build/infrakit-instance-oneandone \
	  -ldflags "-X main.Version=$(VERSION) -X main.Revision=$(REVISION)"

install:
	@echo "+ $@"
	@go install ${GO_LDFLAGS} $(PKGS)

test:
	@echo "+ $@"
	@go test -v
