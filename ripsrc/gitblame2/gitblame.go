package gitblame2

import (
	"os"
	"os/exec"
	"strconv"
	"strings"
)

type Line struct {
	Content    string
	CommitHash string
}

func (l Line) String() string {
	return l.CommitHash + ":" + l.Content
}

type Result struct {
	Lines []Line
}

func (r Result) String() string {
	out := []string{}
	for i, l := range r.Lines {
		out = append(out, strconv.Itoa(i)+":"+l.String())
	}
	return strings.Join(out, "\n")
}

func Run(repoDir, commitHash, file string) (res Result, _ error) {
	args := []string{
		"blame",
		commitHash,
		"--porcelain",
		"--",
		file,
	}
	cmd := exec.Command("git", args...)
	cmd.Dir = repoDir
	cmd.Stderr = os.Stderr
	b, err := cmd.Output()
	if err != nil {
		panic(err)
	}
	res0 := parseOutput(string(b))
	for _, l0 := range res0 {
		l := Line{Content: l0.Content, CommitHash: l0.CommitHash}
		res.Lines = append(res.Lines, l)
	}
	return res, nil
}
