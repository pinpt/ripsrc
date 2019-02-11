package cmdutils

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	"github.com/fatih/color"
)

var ErrRevParseFailed = errors.New("git rev-parse HEAD failed")

func RunOnRepo(ctx context.Context, wr io.Writer, repoDir string, run func() error) error {
	start := time.Now()
	fmt.Fprintf(color.Output, "starting processing repo:%v\n", color.GreenString(repoDir))
	if !hasHeadCommit(ctx, repoDir) {
		fmt.Fprintf(wr, "git rev-parse HEAD failed, happens for empty repos, repo: %v \n", repoDir)
		return ErrRevParseFailed
	}

	err := run()
	if err != nil {
		fmt.Fprintf(color.Output, "completed repo processing in %v repo: %v err: %v\n", time.Since(start), color.RedString(repoDir), color.RedString(err.Error()))

		return err
	}

	fmt.Fprintf(color.Output, "completed repo processing in %v repo: %v\n", time.Since(start), color.GreenString(repoDir))

	return nil
}

func hasHeadCommit(ctx context.Context, repoDir string) bool {
	out := bytes.NewBuffer(nil)
	c := exec.Command("git", "rev-parse", "HEAD")
	c.Dir = repoDir
	c.Stdout = out
	c.Run()
	res := strings.TrimSpace(out.String())
	if len(res) != 40 {
		return false
	}
	return true
}
