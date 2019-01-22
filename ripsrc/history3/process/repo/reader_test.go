package repo

import (
	"os"
	"reflect"
	"testing"

	"github.com/pinpt/ripsrc/ripsrc/pkg/logger"

	"github.com/stretchr/testify/assert"
)

func testReader(t *testing.T) *CheckpointReader {
	return NewCheckpointReader(logger.NewDefaultLogger(os.Stdout))
}

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

	err := testWriter(t).Write(repo, dir, "c1")
	if err != nil {
		t.Fatal(err)
	}

	repo2, err := testReader(t).Read(dir, "")
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(repo, repo2) {
		t.Fatalf("wanted repo %v\ngot repo %v", repo.Debug(), repo2.Debug())
	}
}

func TestReaderValidateCommit(t *testing.T) {
	dir := tempDir()
	defer os.RemoveAll(dir)
	repo := New()
	repo.AddCommit("c1")
	repo["c1"]["p1"] = randomBlameLineLen(1, 1)

	err := testWriter(t).Write(repo, dir, "c1")
	if err != nil {
		t.Fatal(err)
	}

	_, err = testReader(t).Read(dir, "c2")
	if err == nil {
		t.Fatal("expected error with invalid checkpoint commit")
	}
	err2, ok := err.(ErrCheckpointNotExpected)
	if !ok {
		t.Fatal("invalid error type")
	}
	assert.Equal(t, "c2", err2.WantCommit)
	assert.Equal(t, "c1", err2.HaveCommit)
	t.Log("error msg: " + err.Error())
}
