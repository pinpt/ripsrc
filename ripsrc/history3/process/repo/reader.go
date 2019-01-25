package repo

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"time"

	"github.com/pinpt/ripsrc/ripsrc/history3/incblame"
	"github.com/pinpt/ripsrc/ripsrc/history3/process/repo/disk"
	"github.com/pinpt/ripsrc/ripsrc/pkg/logger"
)

type ErrCheckpointNotExpected struct {
	CheckpointDir string
	WantCommit    string
	HaveCommit    string
}

func (s ErrCheckpointNotExpected) Error() string {
	return fmt.Sprintf("ripsrc: requested checkpoint for commit %v, have checkpoint for commit %v", s.WantCommit, s.HaveCommit)
}

type CheckpointReader struct {
	logger logger.Logger
}

func NewCheckpointReader(logger logger.Logger) *CheckpointReader {
	s := &CheckpointReader{}
	s.logger = logger
	return s
}

func (s *CheckpointReader) Read(dir string, expectedCommit string) (Repo, error) {
	dir = filepath.Join(dir, checkpointDirName)

	if expectedCommit != "" {
		// no expected commit validation requested
		b, err := ioutil.ReadFile(filepath.Join(dir, checkpointVersionFile))
		if err != nil {
			return nil, fmt.Errorf("failed reading checkpoint version file, err: %v", err)
		}
		checkpointCommit := string(b)
		if checkpointCommit != expectedCommit {
			return nil, ErrCheckpointNotExpected{CheckpointDir: dir, WantCommit: expectedCommit, HaveCommit: checkpointCommit}
		}
	}

	start := time.Now()
	s.logger.Info("starting reading checkpoint")
	defer func() {
		s.logger.Info("finished reading checkpoint", "dur", time.Since(start))
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
		for {
			obj := &disk.LineData{}
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
	s.logger.Info("loaded line data", "count", len(lineData))
	lines := map[uint64]*incblame.Line{}
	{
		for {
			obj := &disk.Line{}
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
	s.logger.Info("loaded lines", "count", len(lines))
	blames := map[uint64]*incblame.Blame{}
	{
		for {
			obj := &disk.Blame{}
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
	s.logger.Info("loaded unique blames", "count", len(blames))
	{
		i := 0
		for {
			obj := &disk.DataRow{}
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
		s.logger.Info("loaded blames", "count", i)
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
