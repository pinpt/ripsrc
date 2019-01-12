package gitexec

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const cacheDir = "pp-git-cache"
const casheVersion = "1"

func ExecWithCache(ctx context.Context, gitCommand string, repoDir string, args []string) (io.ReadCloser, error) {
	//return ExecPiped(ctx, gitCommand, repoDir, args)

	start := time.Now()
	headCommit := headCommit(ctx, gitCommand, repoDir)
	cacheKey := hashString(casheVersion + "@" + strings.Join(args, "@") + headCommit)[0:32]

	loc := filepath.Join(repoDir, cacheDir, cacheKey+".txt.gz")

	f, err := os.Open(loc)
	if err == nil {
		fmt.Println("Using cache for ", repoDir, strings.Join(args, " "))
		return newGzipFileCloser(f)
	} else {
		if !os.IsNotExist(err) {
			panic(fmt.Errorf("could not open file at location %v, err %v", loc, err))
		}
	}

	err = os.MkdirAll(filepath.Dir(loc), 0777)
	if err != nil {
		return nil, fmt.Errorf("can't create the cache dir for repoDir: '%v' err: %v", repoDir, err)
	}

	err = execToFile(ctx, loc+".tmp", gitCommand, repoDir, args)
	if err != nil {
		return nil, fmt.Errorf("can't write git command output to file repoDir: '%v' err: %v", repoDir, err)
	}

	err = os.Rename(loc+".tmp", loc)
	if err != nil {
		return nil, err
	}

	fmt.Println("Took", time.Since(start), "to run git", repoDir, strings.Join(args, " "))

	f, err = os.Open(loc)
	if err != nil {
		return nil, err
	}
	return newGzipFileCloser(f)

}

type fileCloser struct {
	f *os.File
	io.ReadCloser
}

func (f fileCloser) Close() error {
	err := f.ReadCloser.Close()
	if err != nil {
		return err
	}
	return f.f.Close()
}

func newGzipFileCloser(f *os.File) (io.ReadCloser, error) {
	gr, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}

	res := fileCloser{}
	res.f = f
	res.ReadCloser = gr
	return res, nil
}

func execToFile(ctx context.Context, loc string, gitCommand string, repoDir string, args []string) error {
	f, err := os.Create(loc)
	defer f.Close()
	if err != nil {
		return err
	}

	gw := gzip.NewWriter(f)
	defer gw.Close()

	err = ExecIntoWriter(ctx, gw, gitCommand, repoDir, args)
	if err != nil {
		return err
	}

	return nil
}

func hashString(str string) string {
	h := sha256.Sum256([]byte(str))
	return hex.EncodeToString(h[:])
}

func headCommit(ctx context.Context, gitCommand string, repoDir string) string {
	out := bytes.NewBuffer(nil)
	c := exec.Command(gitCommand, "rev-parse", "HEAD")
	c.Dir = repoDir
	c.Stdout = out
	c.Run()
	res := strings.TrimSpace(out.String())
	if len(res) != 40 {
		return ""
	}
	return res
}

func ExecPiped(ctx context.Context, gitCommand string, repoDir string, args []string) (io.ReadCloser, error) {
	r, wr := io.Pipe()
	go func() {
		err := ExecIntoWriter(ctx, wr, gitCommand, repoDir, args)
		if err != nil {
			panic(err)
		}
		err = wr.Close()
		if err != nil {
			panic(err)
		}
	}()
	return r, nil
}

func Exec(ctx context.Context, gitCommand string, repoDir string, args []string) (io.ReadCloser, error) {
	buf := bytes.NewBuffer(nil)
	err := ExecIntoWriter(ctx, buf, gitCommand, repoDir, args)
	if err != nil {
		return nil, err
	}
	return noopReadCloser{buf}, nil
}

func ExecIntoWriter(ctx context.Context, wr io.Writer, gitCommand string, repoDir string, args []string) error {
	c := exec.CommandContext(ctx, gitCommand, args...)
	c.Dir = repoDir
	c.Stderr = os.Stderr
	c.Stdout = wr
	if err := c.Run(); err != nil {
		return fmt.Errorf("failed executing git command %v", err)
	}
	return nil
}

type noopReadCloser struct {
	io.Reader
}

func (noopReadCloser) Close() error {
	return nil
}
