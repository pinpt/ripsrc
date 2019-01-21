package repo

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/cespare/xxhash"

	"github.com/pinpt/ripsrc/ripsrc/history3/incblame"
	"github.com/pinpt/ripsrc/ripsrc/history3/process/repo/disk"
)

const checkpointDirName = "checkpoint"

func WriteCheckpoint(repo Repo, dir string, lastCommit string) error {
	start := time.Now()
	fmt.Println("starting writing checkpoint")
	defer func() {
		fmt.Println("finished writing checkpoint in", time.Since(start))
	}()
	fmt.Println("preparing to write", len(repo), "commits")

	tmpDir := filepath.Join(dir, "tmp")
	err := os.RemoveAll(tmpDir)
	if err != nil {
		return err
	}

	dir = filepath.Join(dir, checkpointDirName)

	repoWr, err := newMsgWriter(tmpDir, "repo")
	if err != nil {
		return err
	}
	blamesWr, err := newMsgWriter(tmpDir, "blames")
	if err != nil {
		return err
	}
	linesWr, err := newMsgWriter(tmpDir, "lines")
	if err != nil {
		return err
	}
	lineDataWr, err := newMsgWriter(tmpDir, "line-data")
	if err != nil {
		return err
	}

	blamePointerC := uint64(0)
	blamePointers := map[*incblame.Blame]uint64{}

	linePointerC := uint64(0)
	linePointers := map[*incblame.Line]uint64{}

	lineData := map[uint64]bool{}

	writeRepoRow := func(commit string, filePath string, blamePointer uint64) {
		r := &disk.DataRow{}
		r.Commit = commit
		r.Path = filePath
		r.BlamePointer = blamePointer
		err := repoWr.Write(r)
		if err != nil {
			panic(err)
		}
	}

	for ch, commit := range repo {

		for fp, file := range commit {
			if blp, ok := blamePointers[file]; ok {
				writeRepoRow(ch, fp, blp)
				continue
			}
			blamePointerC++
			blp := blamePointerC
			blamePointers[file] = blp

			bl := &sBlame{}
			bl.Pointer = blamePointerC
			bl.Commit = file.Commit
			bl.IsBinary = file.IsBinary
			bl.LinePointers = make([]uint64, 0, len(file.Lines))
			for _, l := range file.Lines {

				if lp, ok := linePointers[l]; ok {
					bl.LinePointers = append(bl.LinePointers, lp)
					continue
				}
				linePointerC++
				lp := linePointerC
				linePointers[l] = lp

				// line data
				dp := xxhash.Sum64(l.Line)
				if ok := lineData[dp]; !ok {
					ld := &sLineData{
						Pointer: dp,
						Data:    l.Line}
					err := lineDataWr.Write(ld)
					if err != nil {
						return err
					}
					lineData[dp] = true
				}

				l2 := &sLine{}
				l2.Pointer = lp
				l2.Commit = l.Commit
				l2.LineDataPointer = dp

				err := linesWr.Write(l2)
				if err != nil {
					return err
				}

				bl.LinePointers = append(bl.LinePointers, lp)
			}

			err := blamesWr.Write(bl)
			if err != nil {
				return err
			}

			writeRepoRow(ch, fp, blp)
		}
	}

	err = repoWr.Finish()
	if err != nil {
		return err
	}
	err = blamesWr.Finish()
	if err != nil {
		return err
	}
	err = linesWr.Finish()
	if err != nil {
		return err
	}
	err = lineDataWr.Finish()
	if err != nil {
		return err
	}

	/*
		err = writeFileAtomic(filepath.Join(tmpDir, "checkpoint-version", []byte(lastCommit) )
		if err != nil {
			return err
		}*/

	err = os.RemoveAll(dir)
	if err != nil {
		return err
	}

	return os.Rename(tmpDir, dir)
}

func writeFileAtomic(loc string, data []byte) error {
	err := ioutil.WriteFile(loc+".tmp", data, 0777)
	if err != nil {
		return err
	}
	return os.Rename(loc+".tmp", loc)
}

type sData struct {
	// map[commitHash]map[filePath]blamePointer
	Data     map[string]map[string]uint64
	Blames   map[uint64]sBlame
	Lines    map[uint64]sLine
	LineData map[uint64]sLineData
}

type sDataRow = disk.DataRow

type sBlame = disk.Blame

type sLine = disk.Line

type sLineData = disk.LineData
