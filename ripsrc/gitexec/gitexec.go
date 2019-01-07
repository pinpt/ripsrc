package gitexec

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"
)

const cacheDir = ".pp-git-cache"

func ExecWithCache(ctx context.Context, gitCommand string, repoDir string, args []string) (io.ReadCloser, error) {
	start := time.Now()
	headCommit := headCommit(ctx, gitCommand, repoDir)
	cacheKey := hashString(strings.Join(args, "@") + headCommit)

	loc := filepath.Join(repoDir, cacheDir, cacheKey+".txt")

	f, err := os.Open(loc)
	if err == nil {
		fmt.Println("Using cache for ", repoDir, strings.Join(args, " "))
		return f, nil
	} else {
		if !os.IsNotExist(err) {
			panic(fmt.Errorf("could not open file at location %v, err %v", loc, err))
		}
	}

	os.MkdirAll(path.Dir(loc), 0777)

	f, err = os.Create(loc + ".tmp")
	defer f.Close()
	if err != nil {
		return nil, err
	}

	err = ExecIntoWriter(ctx, f, gitCommand, repoDir, args)
	if err != nil {
		return nil, err
	}

	f.Close()

	err = os.Rename(loc+".tmp", loc)
	if err != nil {
		return nil, err
	}

	fmt.Println("Took", time.Since(start), "to run git", repoDir, strings.Join(args, " "))

	return os.Open(loc)

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
		panic("invalid head commit sha len")
	}
	return res
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
