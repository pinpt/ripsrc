package ripsrc

//go:generate go run ../genignore.go

import (
	"context"
	"fmt"
	"regexp"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/boyter/scc/processor"
	"github.com/jhaynie/gitblame"
	enry "gopkg.in/src-d/enry.v1"
)

type filejob struct {
	commit   Commit
	filename string
	total    int
	wg       *sync.WaitGroup
}

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

// BlameWorkerPool is worker pool for processing blame
type BlameWorkerPool struct {
	ctx        context.Context
	count      int
	commitjobs chan Commit
	filejobs   chan filejob
	commitdone chan bool
	filedone   chan bool
	errors     chan<- error
	filter     *Filter
	total      int
}

const (
	blacklisted   = "file was on an exclusion list"
	whitelisted   = "file was not on the inclusion list"
	removedFile   = "file was removed"
	limitExceed   = "file size was %dK which exceeds limit of %dK"
	generatedFile = "file was a generated file"
	vendoredFile  = "file was a vendored file"
	configFile    = "file was a config file"
	dotFile       = "file was a dot file"
)

// Start the pool
func (p *BlameWorkerPool) Start() {
	for i := 0; i < p.count; i++ {
		go p.runCommitJobs()
		go p.runFileJobs()
	}
}

// Close the pool and wait for all jobs to complete
func (p *BlameWorkerPool) Close() {
	// close the commit jobs
	close(p.commitjobs)
	// now wait for all the commit jobs to finish before
	// we close the file jobs channel
	for i := 0; i < p.count; i++ {
		<-p.commitdone
	}
	// close the file jobs channel
	close(p.filejobs)
	// now wait for all the file jobs to finish
	for i := 0; i < p.count; i++ {
		<-p.filedone
	}
}

// Submit a job to the worker pool for async processing
func (p *BlameWorkerPool) Submit(job Commit, callback Callback) {
	job.callback = callback
	p.commitjobs <- job
}

func (p *BlameWorkerPool) shouldProcess(filename string) (bool, string) {
	// handle a set of black lists that we should automatically not process
	if enry.IsVendor(filename) {
		return false, vendoredFile
	}
	if enry.IsConfiguration(filename) {
		return false, configFile
	}
	if enry.IsDotFile(filename) {
		return false, dotFile
	}
	if ignorePatterns.MatchString(filename) {
		return false, blacklisted
	}
	return true, ""
}

func (p *BlameWorkerPool) runCommitJobs() {
	defer func() { p.commitdone <- true }()
	for job := range p.commitjobs {
		total := len(job.Files)
		if total > 0 {
			var wg sync.WaitGroup
			for filename, cf := range job.Files {
				var custom bool
				wg.Add(1)
				ok, skipped := p.shouldProcess(filename)
				if ok {
					if p.filter != nil {
						// if a blacklist, exclude if matched
						if p.filter.Blacklist != nil {
							if p.filter.Blacklist.MatchString(filename) {
								skipped = blacklisted
								custom = true
							}
						}
						// if a whitelist, exclude if not matched
						if p.filter.Whitelist != nil {
							if !p.filter.Whitelist.MatchString(filename) {
								skipped = whitelisted
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
						// if not removed (since it won't be in the tree) and the filename
						// looks like a possible license file
						if cf.Status != GitFileCommitStatusRemoved && possibleLicense(filename) {
							buf, err := getBlob(p.ctx, job.Dir, job.SHA, filename)
							if err == nil && !enry.IsBinary(buf) {
								license, err = detect(filename, buf)
							}
							if err != nil {
								job.callback(err, nil, total)
								wg.Done()
								return
							}
						}
						job.callback(nil, &BlameResult{
							Commit:             job,
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
						}, total)
						wg.Done()
					}
				} else {
					// if removed, we need to keep a record so we can detect it
					// but we don't need blame, etc so just send it to the results channel
					if cf.Status == GitFileCommitStatusRemoved {
						job.callback(nil, &BlameResult{
							Commit:             job,
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
						}, total)
						wg.Done()
					} else {
						// only process files that aren't blacklisted
						p.filejobs <- filejob{
							commit:   job,
							filename: filename,
							total:    total,
							wg:       &wg,
						}
					}
				}
			}
			wg.Wait()
			p.total++
			if p.filter != nil && p.filter.Limit > 0 && p.total >= p.filter.Limit {
				return
			}
		} else {
			job.callback(nil, nil, 0)
		}
	}
}

func (p *BlameWorkerPool) runFileJobs() {
	for job := range p.filejobs {
		p.process(job)
	}
	p.filedone <- true
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

func (p *BlameWorkerPool) process(job filejob) {
	defer job.wg.Done()
	lines := make([]*BlameLine, 0)
	// only read in N bytes and ignore the rest
	w := getBuffer()
	defer putBuffer(w)
	var idx int
	var filesize int
	var stopped bool
	// create a callback for blame to track all the author by line
	callback := func(line gitblame.BlameLine) error {
		if stopped || idx >= maxLinePerFile || len(line.Line) >= maxBytesPerLine || len(line.Line)+filesize >= maxFileSize {
			// don't process anymore
			stopped = true
			return nil
		}
		w.WriteString(line.Line)
		w.WriteByte('\n')
		if line.Email == "" {
			line.Email = line.Name
		}
		idx++
		filesize += len(line.Line) + 1 // line feed
		bline := &BlameLine{
			Name:  line.Name,
			Email: line.Email,
			Date:  line.Date,
		}
		// for the first n lines, we include the source so we can try and detect generated files
		if idx < 3 {
			bline.line = &line.Line
		}
		lines = append(lines, bline)
		return nil
	}
	if err := gitblame.GenerateWithContext(p.ctx, job.commit.Dir, job.commit.SHA, job.filename, callback, nil); err != nil {
		if strings.Contains(err.Error(), "exit status 128") {
			// this happens if the path is malformed and the filename cannot be found. long term
			// we should figure out why git doesn't like these filenames even when escaped but
			// it appears to be only when someone checks in a file with a weird character
			// such as src/main\320java/com  .... where \320 is ?
			return
		}
		job.commit.callback(fmt.Errorf("error processing commit %s %s (%s). %v", job.commit.SHA, job.filename, job.commit.Dir, err), nil, job.total)
		return
	}
	// if the file is bigger than what we support, we are going to assume it's a generated file
	if stopped {
		job.commit.callback(nil, &BlameResult{
			Commit:             job.commit,
			Language:           "",
			Filename:           job.filename,
			Lines:              nil,
			Loc:                0,
			Sloc:               0,
			Comments:           0,
			Blanks:             0,
			Complexity:         0,
			WeightedComplexity: 0,
			Skipped:            fmt.Sprintf(limitExceed, w.Len()/1024, maxFileSize/1024),
			Status:             job.commit.Files[job.filename].Status,
		}, job.total)
		return
	}
	buf := w.Bytes()
	language := enry.GetLanguage(job.filename, buf)
	statcallback := &statsProcessor{lines: lines}
	filejob := &processor.FileJob{
		Filename: job.filename,
		Language: language,
		Content:  buf,
		Callback: statcallback,
	}
	processor.CountStats(filejob)
	filejob.Content = nil
	if !statcallback.generated {
		var license *License
		if possibleLicense(job.filename) {
			license, _ = detect(job.filename, buf)
		}
		buf = nil
		job.commit.callback(nil, &BlameResult{
			Commit:             job.commit,
			Language:           language,
			Filename:           job.filename,
			Lines:              lines,
			Size:               int64(filesize),
			Loc:                filejob.Lines,
			Sloc:               filejob.Code,
			Comments:           filejob.Comment,
			Blanks:             filejob.Blank,
			Complexity:         filejob.Complexity,
			WeightedComplexity: filejob.WeightedComplexity,
			License:            license,
			Status:             job.commit.Files[job.filename].Status,
		}, job.total)
	} else {
		buf = nil
		// since we received it, we need to process it ... but this means
		// we stopped processing the file because we detected (below) that
		// it was a generated file ... in this case, we treat it like a
		// deleted file in case it wasn't skipped in a previous commit
		job.commit.callback(nil, &BlameResult{
			Commit:             job.commit,
			Language:           "",
			Filename:           job.filename,
			Lines:              nil,
			Loc:                0,
			Sloc:               0,
			Comments:           0,
			Blanks:             0,
			Complexity:         0,
			WeightedComplexity: 0,
			Skipped:            generatedFile,
			Status:             job.commit.Files[job.filename].Status,
		}, job.total)
	}
	lines = nil
}

// NewBlameWorkerPool returns a new worker pool
func NewBlameWorkerPool(ctx context.Context, errors chan<- error, filter *Filter) *BlameWorkerPool {
	return &BlameWorkerPool{
		ctx:        ctx,
		count:      1,
		commitjobs: make(chan Commit, 10),
		filejobs:   make(chan filejob, runtime.NumCPU()),
		commitdone: make(chan bool, 1),
		filedone:   make(chan bool, 1),
		errors:     errors,
		filter:     filter,
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
	l := p.lines[int(currentLine)-1]
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
