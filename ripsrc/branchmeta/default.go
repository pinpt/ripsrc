package branchmeta

import (
	"context"
	"errors"
	"strings"
)

type Branch struct {
	Name   string
	Commit string
}

func GetDefault(ctx context.Context, repoDir string) (res Branch, _ error) {

	name, err := headBranch(ctx, "git", repoDir)
	if err != nil {
		return res, err
	}
	commit, err := headCommit(ctx, "git", repoDir)
	if err != nil {
		return res, err
	}
	res.Name = name
	res.Commit = commit
	return res, nil
}

func headBranch(ctx context.Context, gitCommand string, repoDir string) (string, error) {
	data, err := execCommand(gitCommand, repoDir, []string{"rev-parse", "--abbrev-ref", "HEAD"})
	if err != nil {
		return "", err
	}
	res := strings.TrimSpace(string(data))
	if res == "HEAD" {
		return "", errors.New("cound not retrieve the name of the default branch")
	}
	return res, nil
}

func headCommit(ctx context.Context, gitCommand string, repoDir string) (string, error) {
	data, err := execCommand(gitCommand, repoDir, []string{"rev-parse", "HEAD"})
	if err != nil {
		return "", err
	}
	res := strings.TrimSpace(string(data))
	if len(res) != 40 {
		return "", errors.New("unexpected output from git rev-parse HEAD")
	}
	return res, nil
}
