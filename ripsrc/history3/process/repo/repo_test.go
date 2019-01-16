package repo

import (
	"os"
	"reflect"
	"testing"
)

func TestBasic1(t *testing.T) {
	td := tempDir()
	defer os.RemoveAll(td)

	repo, err := New(td)
	if err != nil {
		t.Fatal(err)
	}
	blOrig := file("c1",
		line("l1", "c1"))

	add(repo, "c1", "p1", blOrig)
	err = repo.SaveCommit("c1")
	if err != nil {
		t.Fatal(err)
	}

	err = repo.WriteCheckpoint()
	if err != nil {
		t.Fatal(err)
	}

	repo, err = NewFromCheckpoint(td, "c1")
	if err != nil {
		t.Fatal(err)
	}

	files, err := repo.GetFiles("c1")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(files, []string{"p1"}) {
		t.Fatal("file list does not match")
	}

	bl, err := repo.GetFile("c1", "p1")
	if err != nil {
		t.Fatal(err)
	}
	if bl == nil {
		t.Fatal("file p1 not found in c1")
	}
	if !bl.Eq(blOrig) {
		t.Fatal("retrieved blame does not match orig")
	}
}
