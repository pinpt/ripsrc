package repo

import (
	"fmt"
	"path/filepath"
	"reflect"
	"runtime/debug"
	"sync"
	"time"

	"github.com/pinpt/ripsrc/ripsrc/history3/incblame"
	"github.com/pinpt/ripsrc/ripsrc/history3/process/repo/disk"
)

const checkpointFileName = "checkpoint.data"

func WriteCheckpoint(repo Repo, dir string) error {
	start := time.Now()
	fmt.Println("starting writing checkpoint")
	defer func() {
		fmt.Println("finished writing checkpoint in", time.Since(start))
	}()
	data := serializeData(repo)
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
	return msgpWriteToFile(filepath.Join(dir, checkpointFileName), data2)
}

func ReadCheckpoint(dir string) (Repo, error) {
	start := time.Now()
	fmt.Println("starting reading checkpoint")
	defer func() {
		fmt.Println("finished reading checkpoint in", time.Since(start))
	}()

	repo := New()

	data := &disk.Data{}
	err := msgpReadFromFile(filepath.Join(dir, checkpointFileName), data)
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
		if _, ok := repo[file.Commit]; !ok {
			repo[file.Commit] = map[string]*incblame.Blame{}
		}
		bl, ok := blames[file.BlamePointer]
		if !ok {
			panic(bl)
		}
		repo[file.Commit][file.Path] = bl
		i++
	}
	fmt.Println("loaded blames", i)
	return repo, nil
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

func int64p() *int64 {
	v := int64(0)
	return &v
}

var serializeInProgress = 0
var serializePrevGC = 0
var serializeInProgressMu = &sync.Mutex{}

func serializeData(repo Repo) (res sData) {
	// we need to disable gc while in this function, since we rely on pointer addresses being unchanged while here
	serializeInProgressMu.Lock()
	if serializeInProgress == 0 {
		serializePrevGC = debug.SetGCPercent(-1)
	}
	serializeInProgress++
	serializeInProgressMu.Unlock()
	defer func() {
		serializeInProgressMu.Lock()
		serializeInProgress--
		if serializeInProgress == 0 {
			debug.SetGCPercent(serializePrevGC)
		}
		serializeInProgressMu.Unlock()
	}()

	res.Data = map[string]map[string]uint64{}
	res.Blames = map[uint64]sBlame{}
	res.Lines = map[uint64]sLine{}
	res.LineData = map[uint64]sLineData{}

	for ch, commit := range repo {
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
