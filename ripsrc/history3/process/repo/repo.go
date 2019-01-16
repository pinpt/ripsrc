package repo

import (
	"fmt"
	"reflect"

	"github.com/pinpt/ripsrc/ripsrc/history3/incblame"
	"github.com/pinpt/ripsrc/ripsrc/history3/process/repo/disk"
)

type ErrNoCommit struct {
	Commit string
}

func (s ErrNoCommit) Error() string {
	return fmt.Sprintf("commit not found %v", s.Commit)
}

func IsErrNoCommit(err error) bool {
	_, ok := err.(ErrNoCommit)
	return ok
}

type Repo struct {
	dir string

	// map[commitHash]map[filePath]*incblame.Blame
	data map[string]map[string]*incblame.Blame

	// needed for continuation from checkpoint
	fromCheckpoint string
}

func New(checkpointDir string) (*Repo, error) {
	s := &Repo{}
	s.dir = checkpointDir

	s.data = map[string]map[string]*incblame.Blame{}

	return s, nil
}

func NewFromCheckpoint(checkpointDir string, lastProcessedCommit string) (*Repo, error) {
	s, err := New(checkpointDir)
	if err != nil {
		return nil, err
	}
	s.fromCheckpoint = lastProcessedCommit
	return s, nil
}

func (s *Repo) CommitsInMemory() int {
	return len(s.data)
}

func (s *Repo) Add(commitHash, filePath string, blame *incblame.Blame) error {
	if _, ok := s.data[commitHash]; !ok {
		s.data[commitHash] = map[string]*incblame.Blame{}
	}
	s.data[commitHash][filePath] = blame
	return nil
}

func (s *Repo) SaveCommit(commitHash string) error {
	return nil
}

// If commit is not found returns an error.
func (s *Repo) GetFiles(commitHash string) ([]string, error) {
	res := []string{}
	for k := range s.data[commitHash] {
		res = append(res, k)
	}
	return res, nil
}

// GetFile returns file data.
// If commit is not found returns an error.
// If files is not found in commit returns nil for blame and no error.
func (s *Repo) GetFile(commitHash, filePath string) (*incblame.Blame, error) {
	commit, ok := s.data[commitHash]
	if !ok {
		return nil, ErrNoCommit{Commit: commitHash}
	}
	return commit[filePath], nil
}

// Unload removes commit data from memory. Reduces memory use.
func (s *Repo) Unload(commit string) {
	//delete(s.data, commit)
	//delete(s.filesInCommit, commit)
}

func (s *Repo) WriteCheckpoint() error {
	data := s.SerializeData()
	data2 := disk.Data{}
	for ch, commit := range data.Data {
		for fp, b := range commit {
			r := disk.DataRow{}
			r.Commit = ch
			r.Path = fp
			r.BlamePointer = b
			data2.Data = append(data2.Data, r)
		}
	}
	for p, b := range data.Blames {
		r := disk.Blame{}
		r.Pointer = p
		r.Commit = b.Commit
		r.LinePointers = b.LinePointers
		r.IsBinary = b.IsBinary
		data2.Blames = append(data2.Blames, r)
	}
	for p, l := range data.Lines {
		r := disk.Line{}
		r.Pointer = p
		r.Commit = l.Commit
		r.LineDataPointer = l.LineDataPointer
		data2.Lines = append(data2.Lines, r)
	}
	for p, b := range data.LineData {
		r := disk.LineData{}
		r.Pointer = p
		r.Data = b
		data2.LineData = append(data2.LineData, r)
	}
	return nil
}

type sData struct {
	// map[commitHash]map[filePath]blamePointer
	Data     map[string]map[string]uint64
	Blames   map[uint64]sBlame
	Lines    map[uint64]sLine
	LineData map[uint64][]byte
}

type sDataRow struct {
	Commit       string
	Path         string
	BlamePointer uint64
}

type sBlame struct {
	Commit       string
	LinePointers []uint64
	IsBinary     bool
}

type sLine struct {
	Commit          string
	LineDataPointer uint64
}

func (s *Repo) SerializeData() (res sData) {
	return res

	/*
		runtime.GC()
		debug.SetGCPercent(-1)
		defer debug.SetGCPercent(100)

		res.Data = map[string]map[string]uint64{}

		for ch, commit := range s.data {
			res.Data[ch] = map[string]uint64{}
			for fp, file := range commit {
				blp := pointer(file)
				if _, ok := res.Blames[blp]; ok {
					res.Data[ch][fp] = blp
					continue
				}
				bl := sBlame{}
				bl.Commit = file.Commit
				bl.IsBinary = file.IsBinary
				bl.LinePointers = make([]uint64, 0, len(file.Lines))
				for _, l := range file.Lines {
					lp := pointer(l)
					if _, ok := res.Lines[lp]; ok {
						bl.LinePointers = append(bl.LinePointers, lp)
						continue
					}
					dp := pointer(l.Line)
					if _, ok := res.LineData[dp]; !ok {
						res.LineData[dp] = l.Line
					}
					l2 := sLine{}
					l2.Commit = l.Commit
					l2.LineDataPointer = dp
					res.Lines[lp] = l2
					bl.LinePointers = append(bl.LinePointers, lp)
				}
				res.Blames[blp] = bl
				res.Data[ch][fp] = blp
			}
		}

		return res
	*/
}

// could return hash of value instead, but this should be faster
func pointer(v interface{}) uint64 {
	return uint64(reflect.ValueOf(v).Pointer())
}
