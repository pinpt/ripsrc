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
	@GOOS=windows GOARCH=amd64 go build -o $(BUILDDIR)/$(PROG_NAME)-windows

darwin: setup
	@GOOS=darwin GOARCH=amd64 go build -o $(BUILDDIR)/$(PROG_NAME)-darwin

