package ripsrc

import (
	"fmt"
	"regexp"
	"runtime/debug"

	"github.com/boyter/scc/processor"
	"github.com/pinpt/ripsrc/ripsrc/history3/incblame"
	"github.com/pinpt/ripsrc/ripsrc/history3/process"
	enry "gopkg.in/src-d/enry.v1"
)

func (s *Ripper) codeInfoFiles(blame process.Result) (res []BlameResult, _ error) {
	commit := s.commitMeta[blame.Commit]
	for filePath, blf := range blame.Files {
		if filePath == "" {
			fmt.Printf("empty file path, commit %v", commit.SHA)
			continue
		}
		r := BlameResult{}
		r.Filename = filePath
		r.Commit = commit

		f, ok := commit.Files[filePath]
		if !ok {
			panic(fmt.Errorf("Changed file was not found in stats log entry, file %v commit %v", r.Filename, commit.SHA))
		}
		r.Status = f.Status

		if r.Status == GitFileCommitStatusRemoved {
			r.Skipped = removedFile
			// no need to run code info
			res = append(res, r)
			continue
		}

		r, err := s.codeInfoFile(filePath, blf, r)
		if err != nil {
			return nil, err
		}

		res = append(res, r)
	}
	return
}

// maxLinePerFile controls how many lines of code (LOC) we will process before
// determining that it's not a human written source file (generated, etc)
// and skip it
const maxLinePerFile = 40000

// maxBytesPerLine controls the size of one line we will process before
// determining that it's not a human written source file (generated, etc)
// and skip it
const maxBytesPerLine = 1096

// maxFileSize controls the size of the overall file we will process before
// determining that it's not a human written source file (generated, etc)
// and skip it
const maxFileSize = 1000000

const (
	blacklisted        = "File was on an exclusion list"
	whitelisted        = "File was not on the inclusion list"
	removedFile        = "File was removed"
	limitExceed        = "File size was %dK which exceeds limit of %dK"
	maxLineExceed      = "File has more than %d lines"
	maxLineBytesExceed = "File has a line width of >=%dK which is greater than max of %dK"
	generatedFile      = "File was a generated file"
	vendoredFile       = "File was a vendored file"
	configFile         = "File was a config file"
	dotFile            = "File was a dot file"
	pathInvalid        = "File path was invalid"
	languageUnknown    = "Language was unknown"
	fileNotSupported   = "File type was not supported as source code"
	fileBinary         = "File was binary"
)

func (s *Ripper) codeInfoFile(filePath string, bl *incblame.Blame, res BlameResult) (BlameResult, error) {
	fileBytes := blameToFileContent(bl)
	fileSize := len(fileBytes)
	// check for max file size exclusion
	if fileSize >= maxFileSize {
		res.Skipped = fmt.Sprintf(limitExceed, fileSize/1024, maxFileSize/1024)
		return res, nil
	}

	// classify the files language
	language := enry.GetLanguage(filePath, fileBytes)
	if language == "" {
		res.Skipped = languageUnknown
		return res, nil
	}
	res.Language = language

	var lines []*statsLine

	// assign lines to result
	for idx, line := range bl.Lines {
		if idx >= maxLinePerFile {
			res.Skipped = fmt.Sprintf(maxLineExceed, idx)
			return res, nil
		}
		if len(line.Line) >= maxBytesPerLine {
			res.Skipped = fmt.Sprintf(maxLineBytesExceed, len(line.Line)/1024, maxBytesPerLine/1024)
			return res, nil
		}
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

func (s *Ripper) codeStats(filePath string, bl *incblame.Blame, fileBytes []byte, lines []*statsLine, res BlameResult) (BlameResult, error) {
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

	if !statcallback.generated {
		var license *License
		if possibleLicense(filePath) {
			license, _ = detect(filePath, fileBytes)
		}
		res.License = license
	} else {
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
