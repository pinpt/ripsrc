package ripsrc

//go:generate go run ../genignore.go

import (
	"fmt"
	"regexp"
	"runtime/debug"
	"strings"
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
	total  int

	// since commits will often have the same files that we process over and over
	// we can cache the filename exclusion rules to make checking much faster
	// since some of the rule checks are quite expensive regexps, etc.
	hashedExclusions map[string]*exclusionDecision
}

const (
	blacklisted        = "file was on an exclusion list"
	whitelisted        = "file was not on the inclusion list"
	removedFile        = "file was removed"
	limitExceed        = "file size was %dK which exceeds limit of %dK"
	maxLineExceed      = "file has more than %d lines"
	maxLineBytesExceed = "file has a line width of >=%dK which is greater than max of %dK"
	generatedFile      = "file was a generated file"
	vendoredFile       = "file was a vendored file"
	configFile         = "file was a config file"
	dotFile            = "file was a dot file"
	pathInvalid        = "file path was invalid"
	languageUnknown    = "language was unknown"
	fileNotSupported   = "file type was not supported as source code"
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
	// check the cache since some of these lookups are a bit expensive and
	// since filenames within the same repo are usually repeated with many
	// commits and we can reuse the previous decision in subsequent commits
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
	if p.filter != nil && p.filter.Limit > 0 && p.total >= p.filter.Limit {
		return true, nil, nil
	}
	filename := job.file.Name
	cf := job.commit.Files[filename]
	if cf == nil {
		fmt.Println(job.commit.SHA, job.commit.Files)
		panic("commit file was nil for " + job.file.Name)
	}
	if cf.Status == GitFileCommitStatusRemoved { // fast path
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
			Skipped:            "",
			License:            nil,
			Status:             cf.Status,
		}, nil
	}
	p.total++
	var custom bool
	ok, skipped := p.shouldProcess(filename)
	if ok {
		if p.filter != nil {
			// if a blacklist, exclude if matched
			if p.filter.Blacklist != nil {
				if p.filter.Blacklist.MatchString(filename) {
					skipped = blacklisted
					p.hashedExclusions[filename] = &exclusionDecision{false, blacklisted}
					custom = true
				}
			}
			// if a whitelist, exclude if not matched
			if p.filter.Whitelist != nil {
				if !p.filter.Whitelist.MatchString(filename) {
					skipped = whitelisted
					p.hashedExclusions[filename] = &exclusionDecision{false, whitelisted}
					custom = true
				}
			}
		}
	}
	var license *License
	if skipped != "" {
		// if skipped and custom (meaning via filter), we still send it back
		// to indicate that we skipped it
		if !custom {
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
	} else {
		// if removed, we need to keep a record so we can detect it
		// but we don't need blame, etc so just send it to the results channel
		if cf.Status == GitFileCommitStatusRemoved {
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
				License:            license,
				Status:             cf.Status,
			}, nil
		}
	}
	return false, nil, nil
}

// func (p *BlameProcessor) runCommitJobs() {
// 	defer func() { p.commitdone <- true }()
// 	for job := range p.commitjobs {
// 		total := len(job.Files)
// 		// small optimization if there are no files
// 		if total > 0 {
// 			var wg sync.WaitGroup
// 			for filename, cf := range job.Files {
// 				var custom bool
// 				ok, skipped := p.shouldProcess(filename)
// 				if ok {
// 					if p.filter != nil {
// 						// if a blacklist, exclude if matched
// 						if p.filter.Blacklist != nil {
// 							if p.filter.Blacklist.MatchString(filename) {
// 								skipped = blacklisted
// 								p.hashedExclusions[filename] = &exclusionDecision{false, blacklisted}
// 								custom = true
// 							}
// 						}
// 						// if a whitelist, exclude if not matched
// 						if p.filter.Whitelist != nil {
// 							if !p.filter.Whitelist.MatchString(filename) {
// 								skipped = whitelisted
// 								p.hashedExclusions[filename] = &exclusionDecision{false, whitelisted}
// 								custom = true
// 							}
// 						}
// 					}
// 				}
// 				var license *License
// 				if skipped != "" {
// 					// if skipped and custom (meaning via filter), we still send it back
// 					// to indicate that we skipped it
// 					if !custom {
// 						// if not removed (since it won't be in the tree) and the filename
// 						// looks like a possible license file
// 						if cf.Status != GitFileCommitStatusRemoved && possibleLicense(filename) {
// 							buf, err := getBlob(p.ctx, job.Dir, job.SHA, filename)
// 							if err != nil {
// 								job.callback(err, nil, total)
// 								continue
// 							}
// 							if !enry.IsBinary(buf) {
// 								license, err = detect(filename, buf)
// 							}
// 						}
// 						job.callback(nil, &BlameResult{
// 							Commit:             job,
// 							Language:           "",
// 							Filename:           filename,
// 							Lines:              nil,
// 							Loc:                0,
// 							Sloc:               0,
// 							Comments:           0,
// 							Blanks:             0,
// 							Complexity:         0,
// 							WeightedComplexity: 0,
// 							Skipped:            skipped,
// 							License:            license,
// 							Status:             cf.Status,
// 						}, total)
// 					}
// 				} else {
// 					// if removed, we need to keep a record so we can detect it
// 					// but we don't need blame, etc so just send it to the results channel
// 					if cf.Status == GitFileCommitStatusRemoved {
// 						job.callback(nil, &BlameResult{
// 							Commit:             job,
// 							Language:           "",
// 							Filename:           filename,
// 							Lines:              nil,
// 							Loc:                0,
// 							Sloc:               0,
// 							Comments:           0,
// 							Blanks:             0,
// 							Complexity:         0,
// 							WeightedComplexity: 0,
// 							Skipped:            removedFile,
// 							License:            license,
// 							Status:             cf.Status,
// 						}, total)
// 					} else {
// 						// only process files that aren't blacklisted
// 						// don't call wg.Done here because it will be
// 						// called when the file job completes
// 						wg.Add(1)
// 						p.filejobs <- filejob{
// 							commit:   job,
// 							filename: filename,
// 							total:    total,
// 							wg:       &wg,
// 						}
// 					}
// 				}
// 			}
// 			p.total++
// 			// we need to wait for all the file jobs to complete before going to the
// 			// next commit so that they stay ordered
// 			wg.Wait()
// 			if p.filter != nil && p.filter.Limit > 0 && p.total >= p.filter.Limit {
// 				return
// 			}
// 		} else {
// 			job.callback(nil, nil, total)
// 		}
// 	}
// }

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

func (p *BlameProcessor) process(job commitjob) (*BlameResult, error) {
	ok, res, err := p.preprocess(job)
	if ok {
		return res, err
	}
	filename := job.file.Name
	filebuf := []byte(job.file.String())
	filesize := len(filebuf)
	result := &BlameResult{
		Filename: filename,
		Commit:   job.commit,
		Status:   job.commit.Files[filename].Status,
	}
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
		lines = append(lines, &BlameLine{
			Name:  job.commit.Author(),
			Email: job.commit.AuthorEmail,
			Date:  job.commit.Date,
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

	// var idx int
	// var filesize int
	// var stopped bool
	// var reason string
	// // create a callback for blame to track all the author by line
	// callback := func(line gitblame.BlameLine) error {
	// 	if stopped {
	// 		return nil
	// 	}
	// 	if idx >= maxLinePerFile {
	// 		// don't process anymore
	// 		stopped = true
	// 		reason = fmt.Sprintf(maxLineExceed, idx)
	// 		return nil
	// 	}
	// 	if len(line.Line) >= maxBytesPerLine {
	// 		// don't process anymore
	// 		stopped = true
	// 		reason = fmt.Sprintf(maxLineBytesExceed, len(line.Line)/1024, maxBytesPerLine/1024)
	// 		return nil
	// 	}
	// 	if len(line.Line)+filesize >= maxFileSize {
	// 		// don't process anymore
	// 		stopped = true
	// 		reason = fmt.Sprintf(limitExceed, w.Len()/1024, maxFileSize/1024)
	// 		return nil
	// 	}
	// 	if line.Email == "" {
	// 		line.Email = line.Name
	// 	}
	// 	idx++
	// 	filesize += len(line.Line) + 1 // line feed
	// 	bline := &BlameLine{
	// 		Name:  line.Name,
	// 		Email: line.Email,
	// 		Date:  line.Date,
	// 	}
	// 	// for the first n lines, we include the source so we can try and detect generated files
	// 	if idx < 3 {
	// 		bline.line = &line.Line
	// 	}
	// 	lines = append(lines, bline)
	// 	return nil
	// }
	// if err := gitblame.GenerateWithContext(p.ctx, job.commit.Dir, job.commit.SHA, job.filename, callback, nil); err != nil {
	// 	if err == bufio.ErrTooLong {
	// 		// these means we got one too long on a scanned blame line
	// 		job.commit.callback(nil, &BlameResult{
	// 			Commit:             job.commit,
	// 			Language:           "",
	// 			Filename:           job.filename,
	// 			Lines:              nil,
	// 			Loc:                0,
	// 			Sloc:               0,
	// 			Comments:           0,
	// 			Blanks:             0,
	// 			Complexity:         0,
	// 			WeightedComplexity: 0,
	// 			Skipped:            fmt.Sprintf(maxLineBytesExceed, maxBytesPerLine/1024, maxBytesPerLine/1024),
	// 			Status:             job.commit.Files[job.filename].Status,
	// 		}, job.total)
	// 		return
	// 	}
	// 	// on some OS (like windows), blame tries to do something with the file as part of processing
	// 	if strings.Contains(err.Error(), "unsupported filetype") {
	// 		job.commit.callback(nil, &BlameResult{
	// 			Commit:             job.commit,
	// 			Language:           "",
	// 			Filename:           job.filename,
	// 			Lines:              nil,
	// 			Loc:                0,
	// 			Sloc:               0,
	// 			Comments:           0,
	// 			Blanks:             0,
	// 			Complexity:         0,
	// 			WeightedComplexity: 0,
	// 			Skipped:            fileNotSupported,
	// 			Status:             job.commit.Files[job.filename].Status,
	// 		}, job.total)
	// 		return
	// 	}
	// 	// check to see if an invalid file that we can't produce a blame from and then treat this file like it's binary/excluded
	// 	// this happens for files that are commits that are invalid paths that git can't handle such as "www/foobar/\032"
	// 	if strings.Contains(err.Error(), "no such path") {
	// 		job.commit.callback(nil, &BlameResult{
	// 			Commit:             job.commit,
	// 			Language:           "",
	// 			Filename:           job.filename,
	// 			Lines:              nil,
	// 			Loc:                0,
	// 			Sloc:               0,
	// 			Comments:           0,
	// 			Blanks:             0,
	// 			Complexity:         0,
	// 			WeightedComplexity: 0,
	// 			Skipped:            pathInvalid,
	// 			Status:             job.commit.Files[job.filename].Status,
	// 		}, job.total)
	// 		return
	// 	}
	// 	job.commit.callback(fmt.Errorf("error processing commit %s %s (%s). %v", job.commit.SHA, job.filename, job.commit.Dir, err), nil, job.total)
	// 	return
	// }
	// // if the file is bigger than what we support, we are going to assume it's a generated file
	// if stopped {
	// 	job.commit.callback(nil, &BlameResult{
	// 		Commit:             job.commit,
	// 		Language:           "",
	// 		Filename:           job.filename,
	// 		Lines:              nil,
	// 		Loc:                0,
	// 		Sloc:               0,
	// 		Comments:           0,
	// 		Blanks:             0,
	// 		Complexity:         0,
	// 		WeightedComplexity: 0,
	// 		Skipped:            reason,
	// 		Status:             job.commit.Files[job.filename].Status,
	// 	}, job.total)
	// 	return
	// }
	// buf := w.Bytes()
	// language := enry.GetLanguage(job.filename, buf)
	// if language == "" {
	// 	job.commit.callback(nil, &BlameResult{
	// 		Commit:             job.commit,
	// 		Language:           "",
	// 		Filename:           job.filename,
	// 		Lines:              nil,
	// 		Loc:                0,
	// 		Sloc:               0,
	// 		Comments:           0,
	// 		Blanks:             0,
	// 		Complexity:         0,
	// 		WeightedComplexity: 0,
	// 		Skipped:            languageUnknown,
	// 		Status:             job.commit.Files[job.filename].Status,
	// 	}, job.total)
	// 	return
	// }
	// statcallback := &statsProcessor{lines: lines}
	// filejob := &processor.FileJob{
	// 	Filename: job.filename,
	// 	Language: language,
	// 	Content:  buf,
	// 	Callback: statcallback,
	// }
	// processor.CountStats(filejob)
	// filejob.Content = nil
	// if !statcallback.generated {
	// 	var license *License
	// 	if possibleLicense(job.filename) {
	// 		license, _ = detect(job.filename, buf)
	// 	}
	// 	buf = nil
	// 	job.commit.callback(nil, &BlameResult{
	// 		Commit:             job.commit,
	// 		Language:           language,
	// 		Filename:           job.filename,
	// 		Lines:              lines,
	// 		Size:               int64(filesize),
	// 		Loc:                filejob.Lines,
	// 		Sloc:               filejob.Code,
	// 		Comments:           filejob.Comment,
	// 		Blanks:             filejob.Blank,
	// 		Complexity:         filejob.Complexity,
	// 		WeightedComplexity: filejob.WeightedComplexity,
	// 		License:            license,
	// 		Status:             job.commit.Files[job.filename].Status,
	// 	}, job.total)
	// } else {
	// 	buf = nil
	// 	// since we received it, we need to process it ... but this means
	// 	// we stopped processing the file because we detected (below) that
	// 	// it was a generated file ... in this case, we treat it like a
	// 	// deleted file in case it wasn't skipped in a previous commit
	// 	job.commit.callback(nil, &BlameResult{
	// 		Commit:             job.commit,
	// 		Language:           "",
	// 		Filename:           job.filename,
	// 		Lines:              nil,
	// 		Loc:                0,
	// 		Sloc:               0,
	// 		Comments:           0,
	// 		Blanks:             0,
	// 		Complexity:         0,
	// 		WeightedComplexity: 0,
	// 		Skipped:            generatedFile,
	// 		Status:             job.commit.Files[job.filename].Status,
	// 	}, job.total)
	// }
	// lines = nil

	return result, nil
}

// NewBlameProcessor returns a new processor
func NewBlameProcessor(filter *Filter) *BlameProcessor {
	return &BlameProcessor{
		filter:           filter,
		hashedExclusions: make(map[string]*exclusionDecision),
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
