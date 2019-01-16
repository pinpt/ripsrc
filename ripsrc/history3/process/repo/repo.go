package repo

import (
	"fmt"
	"path/filepath"
	"reflect"
	"runtime"
	"runtime/debug"

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

	s.readCheckpoint()

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
	data2 := &disk.Data{}
	for ch, commit := range data.Data {
		for fp, b := range commit {
			r := disk.DataRow{}
			r.Commit = ch
			r.Path = fp
			r.BlamePointer = b
			data2.Data = append(data2.Data, r)
		}
	}
	for _, obj := range data.Blames {
		data2.Blames = append(data2.Blames, obj)
	}
	for _, obj := range data.Lines {
		data2.Lines = append(data2.Lines, obj)
	}
	for _, obj := range data.LineData {
		data2.LineData = append(data2.LineData, obj)
	}
	return msgpWriteToFile(filepath.Join(s.dir, "checkpoint.data"), data2)
}

func (s *Repo) readCheckpoint() error {
	data := &disk.Data{}
	err := msgpReadFromFile(filepath.Join(s.dir, "checkpoint.data"), data)
	if err != nil {
		panic(err)
	}
	lineData := map[uint64][]byte{}
	lines := map[uint64]*incblame.Line{}
	blames := map[uint64]*incblame.Blame{}
	i := 0
	for _, obj := range data.LineData {
		lineData[obj.Pointer] = obj.Data
		i++
	}
	fmt.Println("loaded line data", i)
	i = 0
	for _, obj := range data.Lines {
		line := &incblame.Line{}
		line.Commit = obj.Commit
		v, ok := lineData[obj.LineDataPointer]
		if !ok {
			panic("line data")
		}
		line.Line = v
		lines[obj.Pointer] = line
		i++
	}
	fmt.Println("loaded lines", i)
	i = 0
	for _, obj := range data.Blames {
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
		i++
	}
	fmt.Println("loaded unique blames", i)
	i = 0
	for _, file := range data.Data {
		if _, ok := s.data[file.Commit]; !ok {
			s.data[file.Commit] = map[string]*incblame.Blame{}
		}
		bl, ok := blames[file.BlamePointer]
		if !ok {
			panic(bl)
		}
		s.data[file.Commit][file.Path] = bl
		i++
	}
	fmt.Println("loaded blames", i)
	return nil
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

func (s *Repo) SerializeData() (res sData) {
	runtime.GC()
	debug.SetGCPercent(-1)
	defer debug.SetGCPercent(100)

	res.Data = map[string]map[string]uint64{}
	res.Blames = map[uint64]sBlame{}
	res.Lines = map[uint64]sLine{}
	res.LineData = map[uint64]sLineData{}

	for ch, commit := range s.data {
		res.Data[ch] = map[string]uint64{}
		for fp, file := range commit {
			blp := pointer(file)
			if _, ok := res.Blames[blp]; ok {
				res.Data[ch][fp] = blp
				continue
			}
			bl := sBlame{}
			bl.Pointer = blp
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
					res.LineData[dp] = sLineData{
						Pointer: dp,
						Data:    l.Line}
				}
				l2 := sLine{}
				l2.Pointer = lp
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
}

// could return hash of value instead, but this should be faster
func pointer(v interface{}) uint64 {
	return uint64(reflect.ValueOf(v).Pointer())
}
