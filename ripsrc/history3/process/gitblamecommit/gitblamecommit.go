package gitblamecommit

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/pinpt/ripsrc/ripsrc/gitblame2"
	"github.com/pinpt/ripsrc/ripsrc/history3/incblame"
)

func Blame(ctx context.Context, repoDir string, commitHash string) (map[string]*incblame.Blame, error) {
	files, err := listOfFiles(ctx, repoDir, commitHash)
	if err != nil {
		return nil, err
	}
	res := map[string]*incblame.Blame{}
	for _, p := range files {
		bl, err := gitblameRun(repoDir, commitHash, p)
		if err != nil {
			return nil, err
		}
		res[p] = &bl
	}
	return res, nil
}

func listOfFiles(ctx context.Context, repoDir string, commitHash string) (res []string, _ error) {
	args := []string{
		"ls-tree",
		"--name-only",
		"-r",
		commitHash,
	}
	cmd := exec.Command("git", args...)
	cmd.Dir = repoDir
	cmd.Stderr = os.Stderr
	b, err := cmd.Output()
	if err != nil {
		panic(err)
	}
	for _, l := range strings.Split(strings.TrimSpace(string(b)), "\n") {
		res = append(res, l)
	}
	return res, nil
}

func gitblameRun(repoDir string, commitHash string, filePath string) (res incblame.Blame, _ error) {
	fmt.Println("git blame", repoDir, commitHash, filePath)
	bl, err := gitblame2.Run(repoDir, commitHash, filePath)
	//fmt.Println("running regular blame for file switching from bin mode to regular")
	if err != nil {
		return res, err
	}
	res.Commit = commitHash
	for _, l := range bl.Lines {
		l2 := incblame.Line{}
		l2.Commit = l.CommitHash
		l2.Line = []byte(l.Content)
		res.Lines = append(res.Lines, l2)
	}
	return
}
