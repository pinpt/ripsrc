package repo

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/cespare/xxhash"

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

	metaLoaded    map[string]bool
	filesInCommit map[string][]string
	fileIsLoaded  map[string]map[string]bool
	lastProcessed string

	allLines *store
}

func New(checkpointDir string) (*Repo, error) {
	s := &Repo{}
	s.dir = checkpointDir

	s.data = map[string]map[string]*incblame.Blame{}

	s.metaLoaded = map[string]bool{}
	s.filesInCommit = map[string][]string{}
	s.fileIsLoaded = map[string]map[string]bool{}

	s.allLines = newStore()
	return s, nil
}

func NewFromCheckpoint(checkpointDir string, lastProcessedCommit string) (*Repo, error) {
	s, err := New(checkpointDir)
	if err != nil {
		return nil, err
	}
	s.fromCheckpoint = lastProcessedCommit
	s.allLines, err = newStoreFromFile(s.allLinesLoc(lastProcessedCommit))
	if err != nil {
		return nil, err
	}
	return s, nil
}

const commitSourceProcess = 1
const commitSourceCache = 2

func (s *Repo) commitSource(commitHash string) (byte, error) {
	if s.fromCheckpoint == "" {
		_, ok := s.data[commitHash]
		if !ok {
			return 0, ErrNoCommit{Commit: commitHash}
		}
		return commitSourceProcess, nil
	}
	if s.metaLoaded[commitHash] {
		return commitSourceProcess, nil
	}
	err := s.loadCommitFiles(commitHash)
	if err != nil {
		return 0, err
	}
	return commitSourceCache, nil
}

func (s *Repo) CommitsInMemory() int {
	return len(s.data)
}

func (s *Repo) Add(commitHash, filePath string, blame *incblame.Blame) error {
	s.setData(commitHash, filePath, blame)
	return s.blameWrite(commitHash, filePath, blame)
}

func (s *Repo) setData(commitHash, filePath string, blame *incblame.Blame) {
	_, ok := s.data[commitHash]
	if !ok {
		s.data[commitHash] = map[string]*incblame.Blame{}
	}
	s.data[commitHash][filePath] = blame
}

func (s *Repo) SaveCommit(commitHash string) error {
	var files []string
	commit := s.data[commitHash]
	for file := range commit {
		files = append(files, file)
	}
	obj := &disk.Commit{}
	obj.Files = files

	err := msgpWriteToFile(s.pathForCommit(commitHash), obj)
	if err != nil {
		return err
	}
	s.lastProcessed = commitHash
	return nil
}

func (s *Repo) pathForCommit(commitHash string) string {
	res := filepath.Join(s.dir, "commits", commitHash)
	return res
}

// GetCommit returns the commit data.
// If commit is not found returns an error.
func (s *Repo) GetFiles(commitHash string) ([]string, error) {
	if s.fromCheckpoint == "" {
		res := []string{}
		for k := range s.data[commitHash] {
			res = append(res, k)
		}
		return res, nil
	}
	cs, err := s.commitSource(commitHash)
	if err != nil {
		return nil, err
	}
	if cs == commitSourceProcess {
		res := []string{}
		for k := range s.data[commitHash] {
			res = append(res, k)
		}
		return res, nil
	}
	return s.filesInCommit[commitHash], nil
}

func (s *Repo) loadCommitFiles(commitHash string) error {
	obj := &disk.Commit{}
	err := msgpReadFromFile(s.pathForCommit(commitHash), obj)
	if err != nil {
		return err
	}
	s.filesInCommit[commitHash] = obj.Files
	s.metaLoaded[commitHash] = true
	return nil
}

// GetFile returns file data.
// If commit is not found returns an error.
// If files is not found in commit returns nil for blame and no error.
func (s *Repo) GetFile(commitHash, filePath string) (*incblame.Blame, error) {
	if s.fromCheckpoint == "" {
		commit, ok := s.data[commitHash]
		if !ok {
			return nil, ErrNoCommit{Commit: commitHash}
		}
		return commit[filePath], nil
	}

	// checks that commit exists
	_, err := s.commitSource(commitHash)
	if err != nil {
		return nil, err
	}

	if !s.fileLoaded(commitHash, filePath) {
		bl, err := s.blameRead(commitHash, filePath)
		if err != nil {
			return nil, err
		}
		s.setData(commitHash, filePath, bl)
		s.fileIsLoaded[commitHash][filePath] = true
	}
	commit, ok := s.data[commitHash]
	if !ok {
		return nil, ErrNoCommit{Commit: commitHash}
	}
	return commit[filePath], nil
}

func (s *Repo) fileLoaded(commitHash, filePath string) bool {
	if _, ok := s.fileIsLoaded[commitHash]; !ok {
		s.fileIsLoaded[commitHash] = map[string]bool{}
	}
	return s.fileIsLoaded[commitHash][filePath]
}

// Unload removes commit data from memory. Reduces memory use.
func (s *Repo) Unload(commit string) {
	delete(s.data, commit)
	delete(s.filesInCommit, commit)
}

func (s *Repo) allLinesLoc(lastProcessedCommit string) string {
	loc := filepath.Join(s.dir, "checkpoint", lastProcessedCommit, "all-lines")
	return loc
}

// CheckpointFinalize finishes all pending writes for the checkpoint data.
func (s *Repo) WriteCheckpoint() error {
	checkpointDir := filepath.Join(s.dir, "checkpoint")
	os.RemoveAll(checkpointDir)
	loc := s.allLinesLoc(s.lastProcessed)
	err := os.MkdirAll(filepath.Dir(loc), 0777)
	if err != nil {
		return err
	}
	err = s.allLines.Serialize(loc)
	if err != nil {
		return err
	}
	return nil
}

func (s *Repo) blameWrite(commitHash, filePath string, bl *incblame.Blame) error {
	obj := &disk.BlameData{}
	obj.IsBinary = bl.IsBinary

	for _, line := range bl.Lines {
		lineDataKey := s.allLines.Save(line.Line)
		dl := disk.Line{}
		dl.Commit = line.Commit
		dl.DataKey = uint64(lineDataKey)
		obj.Lines = append(obj.Lines, dl)
	}

	resPath := s.pathForBlame(commitHash, filePath)
	return msgpWriteToFile(resPath, obj)
}

func (s *Repo) pathForBlame(commitHash, filePath string) string {

	filePathKey := xxhash.Sum64String(filePath)
	res := filepath.Join(s.dir, "blames", commitHash, strconv.FormatUint(filePathKey, 10))
	return res
}

func (s *Repo) blameRead(commitHash, filePath string) (*incblame.Blame, error) {
	obj := &disk.BlameData{}
	err := msgpReadFromFile(s.pathForBlame(commitHash, filePath), obj)
	if err != nil {
		return nil, err
	}
	res := &incblame.Blame{}
	res.Commit = commitHash
	for _, l := range obj.Lines {
		l2 := incblame.Line{}
		l2.Commit = l.Commit
		data := s.allLines.Get(storeKey(l.DataKey))
		l2.Line = data
		res.Lines = append(res.Lines, l2)
	}
	return res, nil
}
