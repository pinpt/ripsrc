package ripsrc

import (
	"fmt"
	"io"
	"regexp"
	"runtime/debug"
	"time"

	"github.com/pinpt/ripsrc/ripsrc/fileinfo"

	"github.com/boyter/scc/processor"
	"github.com/pinpt/ripsrc/ripsrc/history3/incblame"
	"github.com/pinpt/ripsrc/ripsrc/history3/process"
)

func (s *Ripsrc) codeInfoFiles(blame process.Result) (res []BlameResult, _ error) {
	commit := s.commitMeta[blame.Commit]

	// check that files are included in both
	files := map[string]bool{}
	for _, cf := range commit.Files {
		if cf.Renamed {
			files[cf.RenamedTo] = true
		}
	}
	for p := range blame.Files {
		files[p] = true
	}
	for p := range files {
		// TODO: double check why
		//if _, ok := commit.Files[p]; !ok {
		//	panic(fmt.Errorf("File was in blame, but not stats output commit:%v path:%v", commit.SHA, p))
		//}
		if _, ok := blame.Files[p]; !ok {
			panic(fmt.Errorf("File was in stats output, but not in blame commit:%v path:%v", commit.SHA, p))
		}
	}

	for filePath, blf := range blame.Files {
		if filePath == "" {
			s.opts.Logger.Info("empty file path", "commit", commit.SHA)
			continue
		}

		r := BlameResult{}
		r.Filename = filePath

		r.Commit = commit

		f, ok := commit.Files[filePath]
		if !ok {
			s.opts.Logger.Debug("changed file was not found in stats log entry", "file", r.Filename, "commit", commit.SHA)
			continue
			panic(fmt.Errorf("Changed file was not found in stats log entry, file %v commit %v", r.Filename, commit.SHA))
		}

		r.Status = f.Status

		if r.Status == GitFileCommitStatusRemoved {
			r.Skipped = removedFile
			// no need to run code info
			res = append(res, r)
			continue
		}

		fileBytes := blameToFileContent(blf)
		fileLines := blameToByteLines(blf)
		info, skipReason := s.fileInfo.GetInfo(fileinfo.InfoArgs{FilePath: filePath, Content: fileBytes, Lines: fileLines})
		r.License = info.License
		r.Language = info.Language

		if skipReason != "" {
			r.Skipped = skipReason
			res = append(res, r)
			continue
		}

		r, err := s.codeInfoFile(filePath, blf, fileBytes, r)
		if err != nil {
			return nil, err
		}

		res = append(res, r)
	}
	return
}

const (
	generatedFile = "file was a generated file"
	//whitelisted      = "File was not on the inclusion list"
	removedFile = "File was removed"
	//pathInvalid      = "File path was invalid"
	//languageUnknown  = "Language was unknown"
	//fileNotSupported = "File type was not supported as source code"
	//fileBinary       = "File was binary"
)

type CodeInfoTimings struct {
	Count int
	Time  time.Duration
}

func (s *CodeInfoTimings) OutputStats(wr io.Writer) {
	fmt.Fprintln(wr, "code info timing")
	fmt.Fprintln(wr, "files processed", s.Count)
	fmt.Fprintln(wr, "total time", s.Time)
}

func (s *Ripsrc) codeInfoFile(filePath string, bl *incblame.Blame, fileBytes []byte, res BlameResult) (BlameResult, error) {
	start := time.Now()
	defer func() {
		dur := time.Since(start)
		s.CodeInfoTimings.Count++
		s.CodeInfoTimings.Time += dur
	}()

	var lines []*statsLine

	// assign lines to result
	for _, line := range bl.Lines {
		meta := s.commitMeta[line.Commit]
		line2 := &statsLine{}
		line2.BlameLine = &BlameLine{}
		line2.Name = meta.AuthorName
		line2.Email = meta.AuthorEmail
		line2.Date = meta.Date
		line2.line = line.Line
		lines = append(lines, line2)
	}

	res, err := s.codeStats(filePath, bl, fileBytes, lines, res)
	if err != nil {
		return res, err
	}

	return res, nil
}

func blameToFileContent(bl *incblame.Blame) (res []byte) {
	for _, l := range bl.Lines {
		res = append(res, l.Line...)
		res = append(res, "\n"...)
	}
	return
}

func blameToByteLines(bl *incblame.Blame) (res [][]byte) {
	for _, l := range bl.Lines {
		res = append(res, l.Line)
	}
	return
}

func (s *Ripsrc) codeStats(filePath string, bl *incblame.Blame, fileBytes []byte, lines []*statsLine, res BlameResult) (BlameResult, error) {
	statcallback := &statsProcessor{lines: lines}
	filejob := &processor.FileJob{
		Filename: filePath,
		Language: res.Language,
		Content:  fileBytes,
		Callback: statcallback,
	}
	processor.CountStats(filejob)
	filejob.Content = nil

	res.Size = int64(len(fileBytes))
	res.Loc = filejob.Lines
	res.Sloc = filejob.Code
	res.Comments = filejob.Comment
	res.Blanks = filejob.Blank
	res.Complexity = filejob.Complexity
	res.WeightedComplexity = filejob.WeightedComplexity

	if statcallback.generated {
		// it was a generated file ... in this case, we treat it like a
		// deleted file in case it wasn't skipped in a previous commit
		res.Language = ""
		res.Skipped = generatedFile
	}

	for _, l := range lines {
		res.Lines = append(res.Lines, l.BlameLine)
	}

	return res, nil
}

type statsLine struct {
	*BlameLine
	line []byte
}

type statsProcessor struct {
	lines     []*statsLine
	generated bool
}

// regular expression to attempt to detect if the file was generated and if so, we exclude it from processing since
// it wasn't written by a human so we don't want to count it in our stats
var generatedRegexp = regexp.MustCompile("(GENERATED|DO NOT EDIT|DO NOT MODIFY|machine generated)")

func (p *statsProcessor) ProcessLine(job *processor.FileJob, currentLine int64, lineType processor.LineType) bool {
	index := int(currentLine) - 1
	if index >= 0 && index < len(p.lines) {
		l := p.lines[index]
		switch lineType {
		case processor.LINE_BLANK:
			l.Blank = true
		case processor.LINE_CODE:
			l.Code = true
		case processor.LINE_COMMENT:
			l.Comment = true
		}
		// if this is a comment and within N lines near the top, we check to see if it
		// has a header that looks like a generated source file
		if l.line != nil && l.Comment {
			src := string(l.line)
			if generatedRegexp.MatchString(src) {
				l.line = nil
				p.generated = true
				return false
			}
			l.line = nil
		}
		return true
	}
	return false
}

func init() {
	processor.DisableCheckBinary = true
	// the ProcessConstants in scc turns off GC since it is mainly used by their cmdline. however, this causes
	// memory leaks. we need to check it and then reset it afterwards.  we first fetch the current value in case
	// it's overriden from the GOGC env.
	currentGC := debug.SetGCPercent(0)
	processor.ProcessConstants()
	// now we need to reset it to the original GC value
	debug.SetGCPercent(currentGC)
}
