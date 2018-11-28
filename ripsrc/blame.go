package ripsrc

//go:generate go run ../genignore.go

import (
	"fmt"
	"regexp"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/boyter/scc/processor"
	"github.com/jhaynie/gitblame"
	enry "gopkg.in/src-d/enry.v1"
)

// BlameLine is a single line entry in blame
type BlameLine struct {
	Name    string
	Email   string
	Date    time.Time
	Comment bool
	Code    bool
	Blank   bool

	// private, only used internally
	line *string
}

// BlameResult holds details about the blame result
type BlameResult struct {
	Commit             Commit
	Language           string
	Filename           string
	Lines              []*BlameLine
	Size               int64
	Loc                int64
	Sloc               int64
	Comments           int64
	Blanks             int64
	Complexity         int64
	WeightedComplexity float64
	Skipped            string
	License            *License
	Status             CommitStatus
}

// tuple for our exclusion check decision
type exclusionDecision struct {
	process bool
	reason  string
}

// BlameProcessor handles processing blame data
type BlameProcessor struct {
	filter *Filter

	// since commits will often have the same files that we process over and over
	// we can cache the filename exclusion rules to make checking much faster
	// since some of the rule checks are quite expensive regexps, etc.
	hashedExclusions map[string]*exclusionDecision
	mu               sync.Mutex

	commitsMetaByHash   map[string]CommitMeta
	commitsMetaByHashMu sync.RWMutex
}

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

func (p *BlameProcessor) isVendored(filename string) bool {
	if enry.IsVendor(filename) {
		// enry will incorrectly match something like:
		// src/com/foo/android/cache/DiskLruCache.java
		// as a vendored file but it's not.... we'll try
		// and correct with heuristics here
		if strings.HasPrefix(filename, "src/") {
			return false
		}
		return true
	}
	return false
}

// handle a set of black lists that we should automatically not process
func (p *BlameProcessor) shouldProcess(filename string) (bool, string) {
	if possibleLicense(filename) {
		return true, ""
	}
	// check the cache since some of these lookups are a bit expensive and
	// since filenames within the same repo are usually repeated with many
	// commits and we can reuse the previous decision in subsequent commits
	p.mu.Lock()
	defer p.mu.Unlock()
	decision := p.hashedExclusions[filename]
	if decision != nil {
		return decision.process, decision.reason
	}
	if enry.IsConfiguration(filename) {
		p.hashedExclusions[filename] = &exclusionDecision{false, configFile}
		return false, configFile
	}
	if enry.IsDotFile(filename) {
		p.hashedExclusions[filename] = &exclusionDecision{false, dotFile}
		return false, dotFile
	}
	if ignorePatterns.MatchString(filename) {
		p.hashedExclusions[filename] = &exclusionDecision{false, blacklisted}
		return false, blacklisted
	}
	if p.isVendored(filename) {
		p.hashedExclusions[filename] = &exclusionDecision{false, vendoredFile}
		return false, vendoredFile
	}
	p.hashedExclusions[filename] = &exclusionDecision{true, ""}
	return true, ""
}

func (p *BlameProcessor) preprocess(job commitjob) (bool, *BlameResult, error) {
	filename := job.filename
	cf := job.commit.Files[filename]
	if cf == nil {
		fmt.Println(job.commit.SHA, job.commit.Files)
		panic("commit file was nil for " + job.file.Name)
	}
	if cf.Status == GitFileCommitStatusRemoved { // fast path
		// if removed, we need to keep a record so we can detect it
		// but we don't need blame, etc so just send it to the results channel
		return true, &BlameResult{
			Commit:             job.commit,
			Language:           "",
			Filename:           filename,
			Lines:              nil,
			Loc:                0,
			Sloc:               0,
			Comments:           0,
			Blanks:             0,
			Complexity:         0,
			WeightedComplexity: 0,
			Skipped:            removedFile,
			License:            nil,
			Status:             cf.Status,
		}, nil
	}
	if cf.Binary { // fast path
		return true, &BlameResult{
			Commit:             job.commit,
			Language:           "",
			Filename:           filename,
			Lines:              nil,
			Loc:                0,
			Sloc:               0,
			Comments:           0,
			Blanks:             0,
			Complexity:         0,
			WeightedComplexity: 0,
			Skipped:            fileBinary,
			License:            nil,
			Status:             cf.Status,
		}, nil
	}
	ok, skipped := p.shouldProcess(filename)
	if ok {
		if p.filter != nil {
			// if a blacklist, exclude if matched
			if p.filter.Blacklist != nil {
				if p.filter.Blacklist.MatchString(filename) {
					skipped = blacklisted
					p.mu.Lock()
					p.hashedExclusions[filename] = &exclusionDecision{false, blacklisted}
					p.mu.Unlock()
				}
			}
			// if a whitelist, exclude if not matched
			if p.filter.Whitelist != nil {
				if !p.filter.Whitelist.MatchString(filename) {
					skipped = whitelisted
					p.mu.Lock()
					p.hashedExclusions[filename] = &exclusionDecision{false, whitelisted}
					p.mu.Unlock()
				}
			}
		}
	}
	var license *License
	if skipped != "" {
		// check if the filename looks like a possible license file
		if possibleLicense(filename) {
			var err error
			buf := []byte(job.file.String())
			if !enry.IsBinary(buf) {
				license, err = detect(filename, buf)
				if err != nil {
					return false, nil, fmt.Errorf("error detecting license for commit %s and file %s. %v", job.commit.SHA, filename, err)
				}
			}
		}
		return true, &BlameResult{
			Commit:             job.commit,
			Language:           "",
			Filename:           filename,
			Lines:              nil,
			Loc:                0,
			Sloc:               0,
			Comments:           0,
			Blanks:             0,
			Complexity:         0,
			WeightedComplexity: 0,
			Skipped:            skipped,
			License:            license,
			Status:             cf.Status,
		}, nil
	}
	return false, nil, nil
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

type CommitMeta struct {
	AuthorName  string
	AuthorEmail string
	Date        time.Time
}

func (p *BlameProcessor) process(job commitjob) (*BlameResult, error) {
	p.commitsMetaByHashMu.RLock()
	if _, ok := p.commitsMetaByHash[job.commit.SHA]; !ok {
		p.commitsMetaByHashMu.RUnlock()
		p.commitsMetaByHashMu.Lock()
		p.commitsMetaByHash[job.commit.SHA] = CommitMeta{
			AuthorName:  job.commit.AuthorName,
			AuthorEmail: job.commit.AuthorEmail,
			Date:        job.commit.Date,
		}
		p.commitsMetaByHashMu.Unlock()
	} else {
		p.commitsMetaByHashMu.RUnlock()
	}

	ok, res, err := p.preprocess(job)
	if ok {
		return res, err
	}
	filename := job.filename
	result := &BlameResult{
		Filename: filename,
		Commit:   job.commit,
		Status:   job.commit.Files[filename].Status,
	}
	if job.file == nil {
		fmt.Println(job.commit, filename)
		panic("file was nil")
	}
	filebuf := []byte(job.file.String())
	filesize := len(filebuf)
	// check for max file size exclusion
	if filesize >= maxFileSize {
		result.Skipped = fmt.Sprintf(limitExceed, filesize/1024, maxFileSize/1024)
		return result, nil
	}

	// classify the files language
	language := enry.GetLanguage(filename, filebuf)
	if language == "" {
		result.Skipped = languageUnknown
		return result, nil
	}
	result.Language = language

	lines := make([]*BlameLine, 0)

	// process our lines into a new struct for handling code classification
	for idx, line := range job.file.Lines {
		if idx >= maxLinePerFile {
			result.Skipped = fmt.Sprintf(maxLineExceed, idx)
			return result, nil
		}
		if len(line.Buffer) >= maxBytesPerLine {
			result.Skipped = fmt.Sprintf(maxLineBytesExceed, len(line.Buffer)/1024, maxBytesPerLine/1024)
			return result, nil
		}

		p.commitsMetaByHashMu.RLock()
		meta, ok := p.commitsMetaByHash[line.Commit]
		if !ok {
			panic("commit metadata not found by sha, were commits processed out of order?")
		}
		p.commitsMetaByHashMu.RUnlock()

		lines = append(lines, &BlameLine{
			Name:  meta.AuthorName,
			Email: meta.AuthorEmail,
			Date:  meta.Date,
			line:  &line.Buffer,
		})
	}
	result.Lines = lines

	statcallback := &statsProcessor{lines: lines}
	filejob := &processor.FileJob{
		Filename: filename,
		Language: language,
		Content:  filebuf,
		Callback: statcallback,
	}
	processor.CountStats(filejob)
	filejob.Content = nil

	if job.file.Empty() && job.commit.Files[filename].Status == GitFileCommitStatusModified {
		fmt.Println("++FILENAME="+filejob.Filename, "sha=", job.commit.SHA, "empty=", job.file.Empty(), "lines=", filejob.Lines, "sloc=", filejob.Code, "filesize=", int64(filesize), "loc=", len(lines))
		panic(10)
	}

	result.Size = int64(filesize)
	result.Loc = filejob.Lines
	result.Sloc = filejob.Code
	result.Comments = filejob.Comment
	result.Blanks = filejob.Blank
	result.Complexity = filejob.Complexity
	result.WeightedComplexity = filejob.WeightedComplexity

	if !statcallback.generated {
		var license *License
		if possibleLicense(filename) {
			license, _ = detect(filename, filebuf)
		}
		result.License = license
	} else {
		// it was a generated file ... in this case, we treat it like a
		// deleted file in case it wasn't skipped in a previous commit
		result.Language = ""
		result.Skipped = generatedFile
	}

	return result, nil
}

// NewBlameProcessor returns a new processor
func NewBlameProcessor(filter *Filter) *BlameProcessor {
	return &BlameProcessor{
		filter:            filter,
		hashedExclusions:  make(map[string]*exclusionDecision),
		commitsMetaByHash: make(map[string]CommitMeta),
	}
}

type statsProcessor struct {
	lines     []*BlameLine
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
			var src = *l.line
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
	// change the default line size for git blame to match our setting
	gitblame.MaxLineSize = maxBytesPerLine
}
