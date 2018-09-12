package ripsrc

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
)

// buffer pool to reduce GC
var bufferPool = sync.Pool{
	// New is called when a new instance is needed
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

// getBuffer fetches a buffer from the pool
func getBuffer() *bytes.Buffer {
	return bufferPool.Get().(*bytes.Buffer)
}

// putBuffer returns a buffer to the pool
func putBuffer(buf *bytes.Buffer) {
	buf.Reset()
	bufferPool.Put(buf)
}

type limitedWriter struct {
	max  int64
	size int64
	buf  bytes.Buffer
}

func (w *limitedWriter) Write(in []byte) (int, error) {
	l := len(in)
	ol := l
	if w.size+int64(l) > w.max {
		r := w.max - w.size
		if r > 0 {
			if r > int64(l) {
				r = int64(l)
			}
			if _, err := w.buf.Write(in[0:r]); err != nil {
				return 0, err
			}
			l = int(r)
		} else {
			l = 0
		}
	} else if l > 0 {
		if _, err := w.buf.Write(in); err != nil {
			return 0, err
		}
	}
	w.size += int64(l)
	return ol, nil
}

func (w *limitedWriter) Bytes() []byte {
	return w.buf.Bytes()
}

func getBlobRef(ctx context.Context, dir string, sha string, filename string) (string, error) {
	buf := getBuffer()
	defer putBuffer(buf)
	cmd := exec.CommandContext(ctx, "git", "ls-tree", sha, "--", filename)
	cmd.Dir = dir
	cmd.Stderr = os.Stderr
	cmd.Stdout = buf
	if err := cmd.Run(); err != nil {
		return "", err
	}
	tok := bytes.Split(buf.Bytes(), space)
	if len(tok) > 2 {
		tok = bytes.Split(tok[2], tab)
		if len(tok) > 1 {
			return strings.TrimSpace(string(tok[0])), nil
		}
	}
	return "", nil
}

func getBlob(ctx context.Context, dir string, sha string, filename string) ([]byte, error) {
	// limit the size of the blob we read in to a bit larger than what linguist wants as max
	// this will prevent reading in huge files that then are rejected anyway OR running
	// out of memory during processing
	ref, err := getBlobRef(ctx, dir, sha, filename)
	if err != nil {
		return nil, fmt.Errorf("error getting blob ref for %s for sha %s (%s)", filename, sha, dir)
	}
	buf := limitedWriter{max: 4092} // only read 4k
	cmd := exec.CommandContext(ctx, "git", "cat-file", "-p", ref)
	cmd.Dir = dir
	cmd.Stderr = os.Stderr
	cmd.Stdout = &buf
	// fmt.Println("running git cat-file", sha)
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	newbuf := make([]byte, buf.size)
	copy(newbuf, buf.buf.Bytes())
	return newbuf, nil
}
