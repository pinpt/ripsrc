package ripsrc

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/boyter/scc/processor"
	"github.com/jhaynie/gitblame"
	enry "gopkg.in/src-d/enry.v1"
)

type filejob struct {
	commit   *Commit
	filename string
}

// BlameLine is a single line entry in blame
type BlameLine struct {
	Name    string
	Email   string
	Date    time.Time
	Comment bool
	Code    bool
	Blank   bool

	line *string
}

// BlameResult holds details about the blame result
type BlameResult struct {
	Commit             *Commit
	Language           string
	Filename           string
	Lines              []*BlameLine
	Loc                int64
	Sloc               int64
	Comments           int64
	Blanks             int64
	Complexity         int64
	WeightedComplexity float64
	Skipped            bool
}

// BlameWorkerPool is worker pool for processing blame
type BlameWorkerPool struct {
	count      int
	commitjobs chan *Commit
	filejobs   chan *filejob
	commitdone chan bool
	filedone   chan bool
	errors     chan<- error
	results    chan<- BlameResult
	filter     *Filter
}

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
	// close the results channel
	close(p.results)
}

// Submit a job to the worker pool for async processing
func (p *BlameWorkerPool) Submit(job *Commit) {
	p.commitjobs <- job
}

var (
	excludeDirs       = regexp.MustCompile("^(\\.git|CVS|\\.svn|\\.hg|Godeps|vendor|node_modules|\\.webpack)\\/")
	excludeFiles      = regexp.MustCompile("^(\\.gitignore|\\.gitattributes|package\\.json|package-lock\\.json|yarn\\.lock|Gopkg\\.lock|Gopkg\\.toml|glide\\.lock|glide\\.yaml|\\.eslintrc|\\.babelrc|\\.eslintignore|\\.travis\\.yml|LICENSE|README|AUTHORS)")
	excludeExtensions = regexp.MustCompile("(?i)\\.(ar|zip|gz|gzip|Z|tar|gif|png|jpg|jpeg|ttf|svg|mpg|mp4|exe|pyc|class|bmp|ico|mov|mp3|pdf|rpm|psd|rtf|tiff|webm|webp|wmv|woff|woff2|xls|xlsx|doc|docx|pptx|ppt|fla|flv|avi|bz2|cab|crx|deb|elf|eot|jxr|lz|midi|otf|swf|bin|pem|p12|pfx|a|o|obj|dylib|dll|so)$")
)

func (p *BlameWorkerPool) shouldProcess(filename string) bool {
	// handle a set of black lists that we should automatically not process
	return !excludeDirs.MatchString(filename) &&
		!excludeExtensions.MatchString(filename) &&
		!excludeFiles.MatchString(filename) &&
		!enry.IsConfiguration(filename) &&
		!enry.IsVendor(filename) &&
		!enry.IsDotFile(filename)
}

func (p *BlameWorkerPool) runCommitJobs() {
	for job := range p.commitjobs {
		if len(job.Files) > 0 {
			for filename, cf := range job.Files {
				if p.shouldProcess(filename) {
					if p.filter != nil {
						// if a blacklist, exclude if matched
						if p.filter.Blacklist != nil {
							if p.filter.Blacklist.MatchString(filename) {
								continue
							}
						}
						// if a whitelist, exclude if not matched
						if p.filter.Whitelist != nil {
							if !p.filter.Whitelist.MatchString(filename) {
								continue
							}
						}
					}
					// if removed, we need to keep a record so we can detect it
					// but we don't need blame, etc so just send it to the results channel
					if cf.Status == GitFileCommitStatusRemoved {
						p.results <- BlameResult{
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
						}
					} else {
						// only process files that aren't blacklisted
						p.filejobs <- &filejob{
							commit:   job,
							filename: filename,
						}
					}
				}
			}
		}
	}
	p.commitdone <- true
}

func (p *BlameWorkerPool) runFileJobs() {
	for job := range p.filejobs {
		p.process(job)
	}
	p.filedone <- true
}

func (p *BlameWorkerPool) process(job *filejob) {
	lines := make([]*BlameLine, 0)
	// only read in N bytes and ignore the rest
	var w strings.Builder
	var idx int
	// create a callback for blame to track all the author by line
	callback := func(line gitblame.BlameLine) error {
		w.WriteString(line.Line)
		w.WriteByte('\n')
		if line.Email == "" {
			line.Email = line.Name
		}
		idx++
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
	if err := gitblame.Generate(job.commit.Dir, job.commit.SHA, job.filename, callback, nil); err != nil {
		if strings.Contains(err.Error(), "exit status 128") {
			// this happens if the path is malformed and the filename cannot be found. long term
			// we should figure out why git doesn't like these filenames even when escaped but
			// it appears to be only when someone checks in a file with a weird character
			// such as src/main\320java/com  .... where \320 is ?
			return
		}
		p.errors <- fmt.Errorf("error processing commit %s %s (%s). %v", job.commit.SHA, job.filename, job.commit.Dir, err)
	}
	buf := []byte(w.String())
	language := enry.GetLanguage(job.filename, buf)
	statcallback := &statsProcessor{lines: lines}
	filejob := &processor.FileJob{
		Filename: job.filename,
		Language: language,
		Content:  buf,
		Callback: statcallback,
	}
	processor.CountStats(filejob)
	buf = nil
	if !statcallback.generated {
		p.results <- BlameResult{
			Commit:             job.commit,
			Language:           language,
			Filename:           job.filename,
			Lines:              lines,
			Loc:                filejob.Lines,
			Sloc:               filejob.Code,
			Comments:           filejob.Comment,
			Blanks:             filejob.Blank,
			Complexity:         filejob.Complexity,
			WeightedComplexity: filejob.WeightedComplexity,
		}
	} else {
		// since we received it, we need to process it ... but this means
		// we stopped processing the file because we detected (below) that
		// it was a generated file ... in this case, we treat it like a
		// deleted file in case it wasn't skipped in a previous commit
		p.results <- BlameResult{
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
			Skipped:            true,
		}
	}
}

// NewBlameWorkerPool returns a new worker pool
func NewBlameWorkerPool(count int, results chan<- BlameResult, errors chan<- error, filter *Filter) *BlameWorkerPool {
	processor.ProcessConstants()
	return &BlameWorkerPool{
		count:      count,
		commitjobs: make(chan *Commit, count*2),
		filejobs:   make(chan *filejob, count*10),
		commitdone: make(chan bool, count),
		filedone:   make(chan bool, count),
		errors:     errors,
		results:    results,
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
			p.generated = true
			return false
		}
	}
	return true
}
