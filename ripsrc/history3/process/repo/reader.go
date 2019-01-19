package repo

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/pinpt/ripsrc/ripsrc/history3/incblame"
	"github.com/pinpt/ripsrc/ripsrc/history3/process/repo/disk"
)

func ReadCheckpoint(dir string) (Repo, error) {
	dir = filepath.Join(dir, checkpointDirName)

	start := time.Now()
	fmt.Println("starting reading checkpoint")
	defer func() {
		fmt.Println("finished reading checkpoint in", time.Since(start))
	}()

	repo := New()

	repoR, err := newMsgReader(dir, "repo")
	if err != nil {
		return nil, err
	}
	blamesR, err := newMsgReader(dir, "blames")
	if err != nil {
		return nil, err
	}
	linesR, err := newMsgReader(dir, "lines")
	if err != nil {
		return nil, err
	}
	lineDataR, err := newMsgReader(dir, "line-data")
	if err != nil {
		return nil, err
	}
	lineData := map[uint64][]byte{}
	{
		obj := &disk.LineData{}
		for {
			err := lineDataR.Read(obj)
			if err != nil {
				if msgIsEOF(err) {
					break
				}
				return nil, err
			}
			lineData[obj.Pointer] = obj.Data
		}
	}
	fmt.Println("loaded line data", len(lineData))
	lines := map[uint64]*incblame.Line{}
	{
		obj := &disk.Line{}
		for {
			err := linesR.Read(obj)
			if err != nil {
				if msgIsEOF(err) {
					break
				}
				return nil, err
			}
			line := &incblame.Line{}
			line.Commit = obj.Commit
			v, ok := lineData[obj.LineDataPointer]
			if !ok {
				panic("line data")
			}
			line.Line = v
			lines[obj.Pointer] = line
		}
	}
	fmt.Println("loaded lines", len(lines))
	blames := map[uint64]*incblame.Blame{}
	{
		obj := &disk.Blame{}
		for {
			err := blamesR.Read(obj)
			if err != nil {
				if msgIsEOF(err) {
					break
				}
				return nil, err
			}
			bl := &incblame.Blame{}
			bl.Commit = obj.Commit
			bl.IsBinary = obj.IsBinary
			for _, lp := range obj.LinePointers {
				line, ok := lines[lp]
				if !ok {
					panic("line")
				}
				bl.Lines = append(bl.Lines, line)
			}
			blames[obj.Pointer] = bl
		}
	}
	fmt.Println("loaded unique blames", len(blames))
	{
		obj := &disk.DataRow{}
		i := 0
		for {
			err := repoR.Read(obj)
			if err != nil {
				if msgIsEOF(err) {
					break
				}
				return nil, err
			}
			bl, ok := blames[obj.BlamePointer]
			if !ok {
				panic("blame")
			}
			if _, ok := repo[obj.Commit]; !ok {
				repo[obj.Commit] = map[string]*incblame.Blame{}
			}
			repo[obj.Commit][obj.Path] = bl
			i++
		}
		fmt.Println("loaded blames", i)
	}

	err = repoR.Finish()
	if err != nil {
		return nil, err
	}
	err = blamesR.Finish()
	if err != nil {
		return nil, err
	}
	err = linesR.Finish()
	if err != nil {
		return nil, err
	}
	err = lineDataR.Finish()
	if err != nil {
		return nil, err
	}

	return repo, nil
}

func msgIsEOF(err error) bool {
	if err.Error() == "unexpected EOF" {
		return true
	}
	return false
}
