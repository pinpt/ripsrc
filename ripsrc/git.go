package ripsrc

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
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

	callback Callback
	diff     *diff
	parent   *Commit
}

func (c Commit) CommitSHA() string {
	return c.SHA
}

func (c Commit) String() string {
	return c.SHA
}

func (c Commit) Author() string {
	if c.AuthorName != "" {
		return c.AuthorName
	}
	return c.AuthorEmail
}

func (c Commit) CommitDate() time.Time {
	return c.Date
}

func (c Commit) IsBinary(filename string) bool {
	return c.Files[filename].Binary
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
		{
			return GitFileCommitStatusAdded
		}
	case "D":
		{
			return GitFileCommitStatusRemoved
		}
	case "M", "R", "C", "MM":
		{
			return GitFileCommitStatusModified
		}
	}
	return GitFileCommitStatusModified
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

// MaxStreamDuration is the maximum duration that a stream should run.
var (
	MaxStreamDuration    = time.Minute * 20
	MaxUnderReadAttempts = 10
	MaxUnderReadDuration = time.Minute
)

// allow our test case to change the executable
var gitCommand, _ = exec.LookPath("git")

type parserState int

const (
	parserStateHeader parserState = iota
	parserStateFiles
	parserStateNumStats
	parserStateDiff
)

type parser struct {
	commits     chan<- Commit
	commit      *Commit
	dir         string
	limit       int
	total       int
	currentDiff *diff
	ordinal     int64
	state       parserState
}

func (p *parser) complete() error {
	if p.currentDiff != nil {
		return p.currentDiff.complete()
	}
	return nil
}

func (p *parser) parse(line string) (bool, error) {
	if line == "" {
		return true, nil
	}
	if Debug {
		fmt.Println(line)
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
				if p.currentDiff != nil {
					if err := p.currentDiff.complete(); err != nil {
						return false, err
					}
				}
				var parent *Commit
				// send the old commit and create a new one
				if p.commit != nil && p.commit.SHA != "" { // because we send when we detect the next commit
					p.commits <- *p.commit
					parent = p.commit
					p.commit = nil
				}
				if p.limit > 0 && p.total >= p.limit {
					p.commit = nil
					return false, nil
				}
				p.currentDiff = newDiffParser()
				p.commit = &Commit{
					Dir:     p.dir,
					SHA:     string(sha),
					Files:   make(map[string]*CommitFile, 0),
					Ordinal: p.ordinal,
					diff:    p.currentDiff,
					parent:  parent,
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
			if bytes.HasPrefix(buf, parentPrefix) {
				parent := string(buf[len(parentPrefix):])
				p.commit.Parent = &parent
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
					p.commit.Files[fn] = &CommitFile{
						Filename: fn,
						Status:   toCommitStatus(action),
					}
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
					p.commit.Files[toFn] = &CommitFile{
						Status:      GitFileCommitStatusAdded,
						Filename:    toFn,
						Renamed:     true,
						RenamedFrom: fromFn,
						RenamedTo:   toFn,
					}
				} else if bytes.HasPrefix(action, copyPrefix) {
					// copy a file into a new file ... it's basically a new file
					fromFn := string(bytes.TrimLeft(paths[0], " "))
					toFn := string(bytes.TrimLeft(paths[1], " "))
					p.commit.Files[toFn] = &CommitFile{
						Status:     GitFileCommitStatusAdded,
						Filename:   toFn,
						Copied:     true,
						CopiedFrom: fromFn,
					}
				} else {
					fn := string(bytes.TrimLeft(paths[0], " "))
					p.commit.Files[fn] = &CommitFile{
						Status:   toCommitStatus(action),
						Filename: fn,
					}
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
			ok, err := p.currentDiff.parse(line)
			// fmt.Println("doing diff", ok, "=>", line)
			if !ok {
				if err != nil {
					return false, err
				}
				p.state = parserStateHeader
				continue
			}
		}
		break
	}
	return true, nil
}

// streamCommits will stream all the commits to the returned channel and block until completed
func streamCommits(ctx context.Context, dir string, sha string, limit int, commits chan<- Commit, errors chan<- error) error {
	gitdir := filepath.Join(dir, ".git")
	if _, err := os.Stat(gitdir); os.IsNotExist(err) {
		return nil
	}
	args := []string{
		"-c", "diff.renameLimit=999999",
		"--no-pager",
		"log",
		"--raw",
		"--reverse",
		"--numstat",
		"--pretty=format:!SHA: %H%n!Committer: %ce%n!CName: %cn%n!Author: %ae%n!AName: %an%n!Signed-Email: %GS%n!Date: %aI%n!Parent: %P%n!Message: %s%n",
		"--no-merges",
		"--full-diff",
		"-p",
		"-m",
		"-U3",
		"-M",
		"-C",
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
	tmpfn := filepath.Join(os.TempDir(), fmt.Sprintf("ripsrc-%v-%d.txt", time.Now().UnixNano(), rand.Int()))
	fn, err := os.Create(tmpfn)
	if err != nil {
		return fmt.Errorf("error creating temp file: %v", err)
	}
	defer os.Remove(fn.Name())
	gitlog := exec.CommandContext(ctx, gitCommand, args...)
	gitlog.Dir = dir
	gitlog.Stdout = fn
	gitlog.Stderr = os.Stderr
	if err := gitlog.Run(); err != nil {
		return fmt.Errorf("error streaming commits from %v. %v", dir, err)
	}
	fn.Close()
	of, err := os.Open(tmpfn)
	if err != nil {
		return fmt.Errorf("error opening temp file output: %v", err)
	}
	defer of.Close()
	var parser parser
	parser.dir = dir
	parser.limit = limit
	parser.commits = commits
	parser.ordinal = time.Now().Unix()
	scanner := bufio.NewScanner(of)
	for scanner.Scan() {
		ok, err := parser.parse(scanner.Text())
		if err != nil {
			return fmt.Errorf("error processing commit from %v. %v", dir, err)
		}
		if !ok {
			break
		}
	}
	if err := parser.complete(); err != nil {
		return err
	}
	if parser.commit != nil && parser.commit.SHA != "" { // because we send when we detect the next commit
		commits <- *parser.commit
	}
	return nil
}

// Debug can be turned off to emit lots of debug info
var Debug = os.Getenv("RIPSRC_GIT_DEBUG") == "true"
