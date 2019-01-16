package repo

import (
	"io/ioutil"

	"github.com/pinpt/ripsrc/ripsrc/history3/incblame"
)

func tempDir() string {
	dir, err := ioutil.TempDir("", "ripsrc-test")
	if err != nil {
		panic(err)
	}
	return dir
}

func file(hash string, lines ...incblame.Line) *incblame.Blame {
	return &incblame.Blame{Commit: hash, Lines: lines}
}

func line(buf string, commit string) incblame.Line {
	return incblame.Line{Line: []byte(buf), Commit: commit}
}

func add(repo *Repo, commitHash, filePath string, blame *incblame.Blame) {
	err := repo.Add(commitHash, filePath, blame)
	if err != nil {
		panic(err)
	}
}
