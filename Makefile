#
# Makefile for building all things related to this repo
#
NAME := ripsrc
ORG := pinpt
PKG := $(ORG)/$(NAME)
PROG_NAME := ripsrc
SHELL := /bin/bash
BASEDIR := $(shell echo $${PWD})
BUILDDIR := $(BASEDIR)/dist
CODESIGN_ID ?= Z54X8R7H9L
GPG_SIGN_ID ?= hello@pinpt.com

.PHONY: clean linux windows darwin generate

all: clean linux windows darwin

dependencies:
	@dep ensure

setup: generate
	@mkdir -p $(BUILDDIR)

clean:
	@rm -rf $(BUILDDIR)

generate:
	@go generate ./ripsrc

linux: setup
	@GOOS=linux GOARCH=amd64 go build -o $(BUILDDIR)/$(PROG_NAME)-linux

windows: setup
	@GOOS=windows GOARCH=amd64 go build -o $(BUILDDIR)/$(PROG_NAME)-windows.exe

darwin: setup codesign
	@GOOS=darwin GOARCH=amd64 go build -o $(BUILDDIR)/$(PROG_NAME)-darwin

tparse:
ifeq (, $(shell which tparse))
	@echo need to install tparse ...
	@go get github.com/mfridman/tparse
endif

test: tparse
	@go test -race -v -cover ./ripsrc -json | tparse -all

# we codesign our OSX binary with Apple Developer Certificate
codesign:
ifeq ($(UNAME_S),Darwin)
ifneq ($(shell security find-identity | grep $(CODESIGN_ID) >/dev/null && echo -n yes),yes)
	$(warning missing Pinpoint Apple Developer Certificate with id $(CODESIGN_ID). Skipping codesign ...)
else
ifneq "$(wildcard $(BUILDDIR)/darwin_amd64/$(PROG_NAME) )" ""
	@echo performing OSX codesign
	@codesign -f --deep -s $(CODESIGN_ID) $(BUILDDIR)/$(PROG_NAME)-darwin
endif
endif
endif
