package ripsrc

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pinpt/ripsrc/ripsrc/patch"
)

// CommitStatus is a commit status type
type CommitStatus string

const (
	// GitFileCommitStatusAdded is the added status
	GitFileCommitStatusAdded = CommitStatus("added")
	// GitFileCommitStatusModified is the modified status
	GitFileCommitStatusModified = CommitStatus("modified")
	// GitFileCommitStatusRemoved is the removed status
	GitFileCommitStatusRemoved = CommitStatus("removed")
)

func (s CommitStatus) String() string {
	return string(s)
}

// CommitFile is a specific detail around a file in a commit
type CommitFile struct {
	Filename    string
	Status      CommitStatus
	Renamed     bool
	Copied      bool
	RenamedFrom string
	RenamedTo   string
	CopiedFrom  string
	Additions   int
	Deletions   int
	Binary      bool
}

// Callback for handling the commit job
type Callback func(err error, result *BlameResult, total int)

// Commit is a specific detail around a commit
type Commit struct {
	Dir            string
	SHA            string
	AuthorName     string
	AuthorEmail    string
	CommitterName  string
	CommitterEmail string
	Files          map[string]*CommitFile
	Date           time.Time
	Ordinal        int64
	Message        string
	Parent         *string
	Signed         bool
	Previous       *Commit

	callback Callback
	diff     *diff
	debug    strings.Builder
}

func (c Commit) String() string {
	return c.SHA
}

// Author returns either the author name (preference) or the email if not found
func (c Commit) Author() string {
	if c.AuthorName != "" {
		return c.AuthorName
	}
	return c.AuthorEmail
}

var (
	commitPrefix        = []byte("!SHA: ")
	authorPrefix        = []byte("!Author: ")
	authorNamePrefix    = []byte("!AName: ")
	committerPrefix     = []byte("!Committer: ")
	committerNamePrefix = []byte("!CName: ")
	signedEmailPrefix   = []byte("!Signed-Email: ")
	messagePrefix       = []byte("!Message: ")
	parentPrefix        = []byte("!Parent: ")
	emailRegex          = regexp.MustCompile("<(.*)>")
	emailBracketsRegex  = regexp.MustCompile("^\\[(.*)\\]$")
	datePrefix          = []byte("!Date: ")
	space               = []byte(" ")
	tab                 = []byte("\t")
	removePrefix        = []byte("R")
	copyPrefix          = []byte("C")
	filenameMask        = regexp.MustCompile("^(100644|100755)$")
	deletedMask         = []byte("000000")
	renameRe            = regexp.MustCompile("(.*)\\{(.*) => (.*)\\}(.*)")
)

func toCommitStatus(name []byte) CommitStatus {
	switch string(name) {
	case "A":
		return GitFileCommitStatusAdded
	case "D":
		return GitFileCommitStatusRemoved
	case "M", "R", "C", "MM", "T":
		return GitFileCommitStatusModified
	}
	panic("unknown commit status: " + string(name))
}

func parseDate(d string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339, d)
	if err != nil {
		return time.Now(), fmt.Errorf("error parsing commit date `%v`. %v", d, err)
	}
	return t.UTC(), nil
}

func parseEmail(email string) string {
	// strip out the angle brackets
	if emailRegex.MatchString(email) {
		m := emailRegex.FindStringSubmatch(email)
		s := m[1]
		// attempt to strip out square brackets if found
		if emailBracketsRegex.MatchString(s) {
			m = emailBracketsRegex.FindStringSubmatch(s)
			return m[1]
		}
		return s
	}
	return ""
}

func getFilename(fn string) (string, string, bool) {
	if renameRe.MatchString(fn) {
		match := renameRe.FindStringSubmatch(fn)
		// use path.Join to remove empty directories and to correct join paths
		// must be path not filepath since it's always unix style in git and on windows
		// filepath will use \
		oldfn := path.Join(match[1], match[2], match[4])
		newfn := path.Join(match[1], match[3], match[4])
		return newfn, oldfn, true
	}
	// straight rename without parts
	s := strings.Split(fn, " => ")
	if len(s) > 1 {
		return s[1], s[0], true
	}
	return fn, fn, false
}

var (
	tabSplitter        = regexp.MustCompile("\\t")
	spaceSplitter      = regexp.MustCompile("[ ]")
	whitespaceSplitter = regexp.MustCompile("\\s+")

	// allow our test case to change the executable
	gitCommand, _ = exec.LookPath("git")
)

func regSplit(text string, splitter *regexp.Regexp) []string {
	indexes := splitter.FindAllStringIndex(text, -1)
	laststart := 0
	result := make([]string, len(indexes)+1)
	for i, element := range indexes {
		result[i] = text[laststart:element[0]]
		laststart = element[1]
	}
	result[len(indexes)] = text[laststart:len(text)]
	return result
}

type commitFileHistory struct {
	gitcachedir string
	mu          sync.RWMutex
	diffs       map[string]*diff
	wg          *sync.WaitGroup
}

func (c *commitFileHistory) GetPreviousCommitSHA(filename string, commit string) string {
	var found string
	c.mu.RLock()
	diff := c.diffs[filename]
	if diff != nil {
		for i, history := range diff.history {
			if history.SHA == commit {
				found = diff.history[i-1].SHA
				break
			}
		}
	}
	c.mu.RUnlock()
	return found
}

func (c *commitFileHistory) getFilename(commit string, filename string) string {
	fn := fmt.Sprintf("%x", sha1.Sum([]byte(filename)))
	return fmt.Sprintf("%s_%s.json.gz", commit, fn)
}

// exists returns true if the file for a given commit already exists
func (c *commitFileHistory) exists(commit string, filename string) bool {
	fn := filepath.Join(c.gitcachedir, c.getFilename(commit, filename))
	if _, err := os.Stat(fn); os.IsNotExist(err) {
		return false
	}
	return true
}

// save the file for a specific commit
func (c *commitFileHistory) save(filename string, commit string, file *patch.File) error {
	fn := filepath.Join(c.gitcachedir, c.getFilename(commit, filename))
	// fmt.Println("saving file", filename, commit, "=>", fn)
	o, err := os.Create(fn)
	if err != nil {
		return fmt.Errorf("error creating cached file at %v. %v", fn, err)
	}
	gz := gzip.NewWriter(o)
	enc := json.NewEncoder(gz)
	enc.Encode(file)
	gz.Flush()
	gz.Close()
	o.Close()
	return nil
}

func (c *commitFileHistory) Get(filename string, commit string) (*patch.File, error) {
	fn := filepath.Join(c.gitcachedir, c.getFilename(commit, filename))
	if _, err := os.Stat(fn); os.IsNotExist(err) {
		f := patch.NewFile(filename)
		// fmt.Println("!!!!!!!!!!! no file found for ", commit, "=>", filename)
		return f, nil
	}
	f, err := os.Open(fn)
	if err != nil {
		return nil, fmt.Errorf("error getting cached file at %v. %v", fn, err)
	}
	r, err := gzip.NewReader(f)
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("error reading cached file at %v. %v", fn, err)
	}
	var pf patch.File
	if err := json.NewDecoder(r).Decode(&pf); err != nil {
		r.Close()
		f.Close()
		return nil, fmt.Errorf("error decoding cached file at %v. %v", fn, err)
	}
	r.Close()
	f.Close()
	return &pf, nil
}

func (c *commitFileHistory) Add(filename string, diff *diff) {
	c.mu.Lock()
	c.diffs[filename] = diff
	c.mu.Unlock()
}

func (c *commitFileHistory) wait() error {
	c.wg.Wait()
	// now we need to process them all
	processed := make(map[string]bool)
	var err error
	for filename, diff := range c.diffs {
		processed, err = c.process(filename, diff, processed)
		if err != nil {
			return err
		}
	}
	c.diffs = nil
	return nil
}

func (c *commitFileHistory) process(filename string, diff *diff, processed map[string]bool) (map[string]bool, error) {
	var err error
	for i, history := range diff.history {
		if history.Binary {
			// ignore binary changes
			continue
		}
		if c.exists(history.SHA, filename) {
			continue // commits are idompotent so if we've already processed it, just use it again
		}
		var file *patch.File
		// fmt.Println("$$$", filename, i, history.SHA, "copy="+history.CopyFrom, "rename="+history.RenameFrom, "deleted=", history.Deleted)
		if history.CopyFrom != "" {
			// copy from, so we need to process
			if !processed[history.CopyFrom] {
				// not processed, we need to preprocess
				for fn, d := range c.diffs {
					if fn == history.CopyFrom {
						processed, err = c.process(fn, d, processed)
						if err != nil {
							return nil, err
						}
						processed[fn] = true
					}
				}
			}
			// get the copied file buffer
			previousSHA := c.GetPreviousCommitSHA(history.CopyFrom, history.SHA)
			if previousSHA == "" {
				panic("couldn't find previous sha for " + history.CopyFrom + " from commit " + history.SHA)
			}
			existingFile, err := c.Get(history.CopyFrom, previousSHA)
			if err != nil {
				return nil, err
			}
			file = patch.NewFile(filename)
			if err := file.Parse(existingFile.String(), history.SHA); err != nil {
				return nil, fmt.Errorf("error processing copied file %v => %s for commit %v. %v", history.CopyFrom, filename, history.SHA, err)
			}
			// fmt.Println(file.Stringify(true))
			// panic(history.Patch.String())
		} else if history.RenameFrom != "" {
			// copy from, so we need to process
			if !processed[history.RenameFrom] {
				// not processed, we need to preprocess
				for fn, d := range c.diffs {
					if fn == history.RenameFrom {
						processed, err = c.process(fn, d, processed)
						if err != nil {
							return nil, err
						}
						processed[fn] = true
					}
				}
			}
			// get the copied file buffer
			previousSHA := c.GetPreviousCommitSHA(history.RenameFrom, history.SHA)
			if previousSHA == "" {
				panic("couldn't find previous sha for " + history.RenameFrom + " from commit " + history.SHA)
			}
			existingFile, err := c.Get(history.RenameFrom, previousSHA)
			if err != nil {
				return nil, err
			}
			file = patch.NewFile(history.RenameFrom)
			if err := file.Parse(existingFile.String(), history.SHA); err != nil {
				return nil, fmt.Errorf("error processing renamed file %v => %s for commit %v. %v", history.RenameFrom, filename, history.SHA, err)
			}
		}
		if file == nil {
			if i > 0 {
				previousha := diff.history[i-1].SHA
				file, err = c.Get(filename, previousha)
				if err != nil {
					return nil, err
				}
			} else {
				file = patch.NewFile(filename)
			}
		}
		// if this is an empty file and the patch is for a merge, skip it
		if history.Patch.MergeCommit {
			continue
		}
		// if we have an empty file and it's not new, this is related to a merge
		if !history.NewFile && file.Empty() {
			continue
		}
		if history.Patch.Empty() {
			continue
		}
		if !history.Deleted {
			// fmt.Println("#############", filename, history.SHA, "BEFORE >>"+file.Stringify(true)+"<<")
			// fmt.Println("PATCH", filename, ">>", history.Patch.String()+"<<")
			newfile := history.Patch.Apply(file, history.SHA)
			// fmt.Println(file, history.SHA, "AFTER >>"+newfile.Stringify(true)+"<<")
			if err := c.save(filename, history.SHA, newfile); err != nil {
				return nil, err
			}
		}
	}
	processed[filename] = true
	return processed, nil
}

func newCommitFileHistory(gitcachedir string) *commitFileHistory {
	var wg sync.WaitGroup
	wg.Add(1)
	return &commitFileHistory{
		wg:          &wg,
		diffs:       make(map[string]*diff),
		gitcachedir: gitcachedir,
	}
}

type fileprocessor struct {
	ctx            context.Context
	dir            string
	files          chan *CommitFile
	wg             sync.WaitGroup
	errors         chan<- error
	history        *commitFileHistory
	blameProcessor *BlameProcessor
}

func (p *fileprocessor) wait() {
	p.wg.Wait()
}

func (p *fileprocessor) close() {
	close(p.files)
	p.wg.Wait()
}

func (p *fileprocessor) process(filename string) error {
	// fmt.Println("$$$$ processing=" + filename)
	args := []string{
		"-c", "diff.renameLimit=999999",
		"log",
		"-p",
		"--reverse",
		"--no-abbrev-commit",
		"--pretty=format:!SHA: %H%n!Parent: %P",
		"-m",
		"--first-parent",
		"--",
		filename,
	}
	c := exec.CommandContext(p.ctx, gitCommand, args...)
	var stdout bytes.Buffer
	c.Dir = p.dir
	c.Stderr = os.Stderr
	c.Stdout = &stdout
	if err := c.Run(); err != nil {
		return fmt.Errorf("error fetching file details for %v. %v", filename, err)
	}
	dp := newDiffParser(filename)
	s := bufio.NewScanner(&stdout)
	for s.Scan() {
		// fmt.Println("******* ", s.Text())
		ok, err := dp.parse(s.Text())
		if err != nil {
			return fmt.Errorf("error scanning for file details for %v. %v", filename, err)
		}
		if !ok {
			break
		}
	}
	if err := dp.complete(); err != nil {
		return fmt.Errorf("error parsing diff details for %v. %v", filename, err)
	}
	p.history.Add(filename, dp)
	return nil
}

func (p *fileprocessor) run() {
	processed := make(map[string]bool)
	var mu sync.Mutex
	for i := 0; i < runtime.NumCPU(); i++ {
		p.wg.Add(1)
		go func() {
			defer p.wg.Done()
			for cf := range p.files {
				filename := cf.Filename
				var needToProcess bool
				mu.Lock()
				if !processed[filename] {
					processed[filename] = true
					needToProcess = true
				}
				mu.Unlock()
				if needToProcess {
					if cf.Binary {
						// don't need to process binary files
						continue
					}
					if ok, _ := p.blameProcessor.shouldProcess(filename); !ok {
						// quick path to skip obvious files we'll skip later
						continue
					}
					if err := p.process(filename); err != nil {
						p.errors <- err
						return
					}
				}
			}
		}()
	}
}

type parserState int

const (
	parserStateHeader parserState = iota
	parserStateFiles
	parserStateNumStats
	parserStateDiff
)

type parser struct {
	commits  chan<- Commit
	filejobs chan<- *CommitFile
	commit   *Commit
	dir      string
	limit    int
	total    int
	ordinal  int64
	state    parserState
}

func (p *parser) parse(line string) (bool, error) {
	if line == "" {
		return true, nil
	}
	if Debug {
		fmt.Println(line)
	}
	if RipDebug {
		defer func(line string) {
			if p.commit != nil {
				p.commit.debug.WriteString(line)
				p.commit.debug.WriteString("\n")
			}
		}(line)
	}
	buf := []byte(line)
	for {
		switch p.state {
		case parserStateHeader:
			// fmt.Println(line)
			if bytes.HasPrefix(buf, commitPrefix) {
				sha := string(buf[len(commitPrefix):])
				i := strings.Index(sha, " ")
				if i > 0 {
					// trim off stuff after the sha since we can get tag info there
					sha = sha[0:i]
				}
				var parent *string
				if p.commit != nil {
					parent = &p.commit.SHA
				}
				var parentCommit *Commit
				// send the old commit and create a new one
				if p.commit != nil && p.commit.SHA != "" { // because we send when we detect the next commit
					parentCommit = p.commit
					p.commits <- *p.commit
					p.commit = nil
				}
				if p.limit > 0 && p.total == p.limit {
					p.commit = nil
					return false, nil
				}
				p.commit = &Commit{
					Dir:      p.dir,
					SHA:      string(sha),
					Files:    make(map[string]*CommitFile, 0),
					Ordinal:  p.ordinal,
					Parent:   parent,
					Previous: parentCommit,
				}
				p.ordinal++
				p.total++
				return true, nil
			}
			if bytes.HasPrefix(buf, datePrefix) {
				d := bytes.TrimSpace(buf[len(datePrefix):])
				t, err := parseDate(string(d))
				if err != nil {
					return false, fmt.Errorf("error parsing commit %s in %s. %v", p.commit.SHA, p.dir, err)
				}
				p.commit.Date = t.UTC()
				return true, nil
			}
			if bytes.HasPrefix(buf, authorPrefix) {
				p.commit.AuthorEmail = string(buf[len(authorPrefix):])
				return true, nil
			}
			if bytes.HasPrefix(buf, authorNamePrefix) {
				p.commit.AuthorName = string(buf[len(authorNamePrefix):])
				return true, nil
			}
			if bytes.HasPrefix(buf, committerPrefix) {
				p.commit.CommitterEmail = string(buf[len(committerPrefix):])
				return true, nil
			}
			if bytes.HasPrefix(buf, committerNamePrefix) {
				p.commit.CommitterName = string(buf[len(committerNamePrefix):])
				return true, nil
			}
			if bytes.HasPrefix(buf, signedEmailPrefix) {
				signedCommitLine := string(buf[len(signedEmailPrefix):])
				if signedCommitLine != "" {
					p.commit.Signed = true
					signedEmail := parseEmail(signedCommitLine)
					if signedEmail != "" {
						// if signed, mark it as such as use this as the preferred email
						p.commit.AuthorEmail = signedEmail
					}
				}
				return true, nil
			}
			if bytes.HasPrefix(buf, messagePrefix) {
				p.commit.Message = string(buf[len(messagePrefix):])
				p.state = parserStateFiles
				return true, nil
			}
		case parserStateFiles:
			if buf[0] == ':' {
				// fmt.Println(string(buf))
				// :100644␠100644␠d1a02ae0...␠a452aaac...␠M␉·pandora/pom.xml
				tok1 := bytes.Split(buf, space)
				mask := tok1[1]
				// fmt.Println(p.commit.SHA, line, string(mask))
				// if the mask isn't a regular file or deleted file, skip it
				if !filenameMask.Match(mask) && !bytes.Equal(mask, deletedMask) {
					return true, nil
				}
				tok2 := bytes.Split(bytes.Join(tok1[4:], space), tab)
				action := tok2[0]
				paths := tok2[1:]
				if len(action) == 1 {
					fn := string(bytes.TrimLeft(paths[0], " "))
					cf := &CommitFile{
						Filename: fn,
						Status:   toCommitStatus(action),
					}
					p.commit.Files[fn] = cf
					p.filejobs <- cf
				} else if bytes.HasPrefix(action, removePrefix) {
					// rename
					fromFn := string(bytes.TrimLeft(paths[0], " "))
					toFn := string(bytes.TrimLeft(paths[1], " "))
					// panic("renaming [" + fromFn + "] => [" + toFn + "]")
					p.commit.Files[fromFn] = &CommitFile{
						Status:      GitFileCommitStatusRemoved,
						Filename:    fromFn,
						Renamed:     true,
						RenamedFrom: fromFn,
						RenamedTo:   toFn,
					}
					cf := &CommitFile{
						Status:      GitFileCommitStatusAdded,
						Filename:    toFn,
						Renamed:     true,
						RenamedFrom: fromFn,
						RenamedTo:   toFn,
					}
					p.commit.Files[toFn] = cf
					p.filejobs <- cf
				} else if bytes.HasPrefix(action, copyPrefix) {
					// copy a file into a new file ... it's basically a new file
					fromFn := string(bytes.TrimLeft(paths[0], " "))
					toFn := string(bytes.TrimLeft(paths[1], " "))
					cf := &CommitFile{
						Status:     GitFileCommitStatusAdded,
						Filename:   toFn,
						Copied:     true,
						CopiedFrom: fromFn,
					}
					p.commit.Files[toFn] = cf
					p.filejobs <- cf
				} else {
					fn := string(bytes.TrimLeft(paths[0], " "))
					cf := &CommitFile{
						Status:   toCommitStatus(action),
						Filename: fn,
					}
					p.commit.Files[fn] = cf
					p.filejobs <- cf
				}
				return true, nil
			}
			p.state = parserStateNumStats
			continue
		case parserStateNumStats:
			tok := bytes.Split(buf, tab)
			// handle the file stats output
			if len(tok) == 3 {
				tok := regSplit(string(buf), tabSplitter)
				fn, _, _ := getFilename(tok[2])
				file := p.commit.Files[fn]
				// fmt.Println("num stats", fn, "=>", oldfn, "renamed", renamed, "is nil?", file == nil)
				if file == nil {
					// this is OK, just means it was a special entry such as directory only, skip this one
					return true, nil
				}
				if tok[0] == "-" {
					file.Binary = true
				} else {
					adds, _ := strconv.ParseInt(tok[0], 10, 32)
					dels, _ := strconv.ParseInt(tok[1], 10, 32)
					file.Additions = int(adds)
					file.Deletions = int(dels)
				}
			} else {
				p.state = parserStateDiff
				continue
			}
		case parserStateDiff:
			if line[0:1] == "!" {
				p.state = parserStateHeader
				continue
			}
		}
		break
	}
	return true, nil
}

// streamCommits will stream all the commits to the returned channel and block until completed
func streamCommits(ctx context.Context, dir string, cachedir string, sha string, limit int, blameProcessor *BlameProcessor, history *commitFileHistory, commits chan<- Commit, errors chan<- error) error {
	cachefn := filepath.Join(cachedir, "gitlog.txt")
	cacheshafn := filepath.Join(cachedir, "gitlog_sha.txt")
	var of io.ReadCloser
	if sha == "" {
		if _, err := os.Stat(cachefn); err == nil {
			if _, err := os.Stat(cacheshafn); err == nil {
				buf, _ := ioutil.ReadFile(cacheshafn)
				if buf != nil && len(buf) > 0 {
					var out strings.Builder
					c := exec.CommandContext(ctx, gitCommand, "rev-parse", "HEAD")
					c.Stdout = &out
					c.Run()
					if out.Len() > 0 && string(buf) == strings.TrimSpace(out.String()) {
						of, _ = os.Open(cachefn)
					}
				}
			}
		}
	}
	if of == nil {
		args := []string{
			"-c", "diff.renameLimit=999999",
			"--no-pager",
			"log",
			"--raw",
			"--reverse",
			"--numstat",
			"--pretty=format:!SHA: %H%n!Committer: %ce%n!CName: %cn%n!Author: %ae%n!AName: %an%n!Signed-Email: %GS%n!Date: %aI%n!Message: %s%n",
			"--no-merges",
			"--topo-order",
		}
		// if provided, we need to start streaming after this commit forward
		if sha != "" {
			args = append(args, sha+"...")
		}
		if Debug {
			fmt.Println(dir, gitCommand, strings.Join(args, " "))
		}
		// stream to a temp file and then re-read it in ... seems to be way more stable
		f, err := os.Create(cachefn)
		if err != nil {
			return fmt.Errorf("error creating temp file: %v", err)
		}
		gitlog := exec.CommandContext(ctx, gitCommand, args...)
		gitlog.Dir = dir
		gitlog.Stdout = f
		gitlog.Stderr = os.Stderr
		if err := gitlog.Run(); err != nil {
			return fmt.Errorf("error streaming commits from %v. %v", dir, err)
		}
		f.Close() // close and re-open
		of, err = os.Open(cachefn)
		if err != nil {
			return fmt.Errorf("error opening temp file %v for output: %v", cachefn, err)
		}
	}
	defer of.Close()

	filejobs := make(chan *CommitFile, 100)
	localerrors := make(chan error, 1)
	finalerror := make(chan error, 1)

	var parser parser
	parser.dir = dir
	parser.limit = limit
	parser.commits = commits
	parser.ordinal = time.Now().Unix()
	parser.filejobs = filejobs

	var processor fileprocessor
	processor.files = filejobs
	processor.dir = dir
	processor.ctx = ctx
	processor.errors = localerrors
	processor.history = history
	processor.blameProcessor = blameProcessor
	processor.run()
	scanner := bufio.NewScanner(of)

	defer history.wg.Done()

	go func() {
		for err := range localerrors {
			processor.close()
			of.Close()
			finalerror <- err
			break
		}
	}()

	for scanner.Scan() {
		ok, err := parser.parse(scanner.Text())
		if err != nil {
			return fmt.Errorf("error processing commit from %v. %v", dir, err)
		}
		if !ok {
			break
		}
	}
	if parser.commit != nil && parser.commit.SHA != "" { // because we send when we detect the next commit
		commits <- *parser.commit
	}
	processor.close()
	select {
	case err := <-finalerror:
		return err
	default:
		break
	}
	if parser.commit != nil && parser.commit.SHA != "" {
		ioutil.WriteFile(cacheshafn, []byte(parser.commit.SHA), 0644)
	}
	return nil
}

// Debug can be turned off to emit lots of debug info
var Debug = os.Getenv("RIPSRC_GIT_DEBUG") == "true"
