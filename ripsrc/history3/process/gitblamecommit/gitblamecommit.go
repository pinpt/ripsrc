package gitblamecommit

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"

	"github.com/pinpt/ripsrc/ripsrc/gitblame2"
	"github.com/pinpt/ripsrc/ripsrc/history3/incblame"
)

var concurrency = 2 * runtime.NumCPU()

// Another approach to running blames on demand only
// Could lead to unpredictable performance.
// Would also lose changes in merges.
func Blame2(ctx context.Context, repoDir string, commitHash string) (map[string]*incblame.Blame, error) {
	files, err := listOfFiles(ctx, repoDir, commitHash)
	if err != nil {
		return nil, err
	}
	res := map[string]*incblame.Blame{}
	for _, p := range files {
		// this is all that is needed, because in processing if we see that parent is binary, but patch is not, we run git blame to get real info
		bl := &incblame.Blame{}
		bl.IsBinary = true
		res[p] = bl
	}
	return res, nil
}

func Blame(ctx context.Context, repoDir string, commitHash string) (map[string]*incblame.Blame, error) {
	files, err := listOfFiles(ctx, repoDir, commitHash)
	if err != nil {
		return nil, err
	}
	filesChan := stringsChan(files)
	res := map[string]*incblame.Blame{}
	resMu := sync.Mutex{}
	wg := sync.WaitGroup{}
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for p := range filesChan {
				bl, err := gitblameRun(repoDir, commitHash, p)
				if err != nil {
					panic(err)
				}
				resMu.Lock()
				res[p] = &bl
				resMu.Unlock()
			}
		}()
	}
	wg.Wait()
	return res, nil
}

func stringsChan(arr []string) chan string {
	res := make(chan string)
	go func() {
		for _, s := range arr {
			res <- s
		}
		close(res)
	}()
	return res
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
