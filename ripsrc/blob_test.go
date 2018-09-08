package ripsrc

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBlob(t *testing.T) {
	assert := assert.New(t)
	cwd, _ := os.Getwd()
	dir := filepath.Join(cwd, "..")
	buf, err := getBlob(context.Background(), dir, "a0ec861c9e157908d06a814079abbb63e54784e3", "Makefile")
	assert.NoError(err)
	assert.NotEmpty(buf)
	assert.Contains(string(buf), "Makefile for building all things related to this repo")
}

func TestBlobRef(t *testing.T) {
	assert := assert.New(t)
	cwd, _ := os.Getwd()
	dir := filepath.Join(cwd, "..")
	buf, err := getBlobRef(context.Background(), dir, "a0ec861c9e157908d06a814079abbb63e54784e3", "Makefile")
	assert.NoError(err)
	assert.NotEmpty(buf)
	assert.Equal("5a35150578be18d8c3b971ad06e968e1f14529dd", string(buf))
}
