package repo

import (
	"os"
	"reflect"
	"testing"
)

func TestReaderBasic1(t *testing.T) {
	dir := tempDir()
	defer os.RemoveAll(dir)

	repo := New()

	for i := 0; i < 2; i++ {
		ch := randomString(32)
		repo.AddCommit(ch)
		for i := 0; i < 2; i++ {
			fp := randomString(1)
			repo[ch][fp] = randomBlameLineLen(1, 2)
		}
	}

	err := WriteCheckpoint(repo, dir, "c1")
	if err != nil {
		t.Fatal(err)
	}

	repo2, err := ReadCheckpoint(dir)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(repo, repo2) {
		t.Fatalf("wanted repo %v\ngot repo %v", repo.Debug(), repo2.Debug())
	}
}
